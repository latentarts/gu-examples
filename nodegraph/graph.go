//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/latentart/gu/jsutil"
)

// ── Constants ──────────────────────────────────────────────────────────

const (
	nodeW    = 180
	headerH  = 36
	portRowH = 28
	portPadY = 12
	portR    = 6
	minZoom  = 0.15
	maxZoom  = 3.0
)

// ── Types ──────────────────────────────────────────────────────────────

type nodeData struct {
	ID   string
	Type string
	X, Y float64
}

type connData struct {
	FromNode string
	FromPort int
	ToNode   string
	ToPort   int
}

type nodeTypeDef struct {
	Name    string
	Color   string
	Icon    string
	Inputs  []string
	Outputs []string
}

// ── Catalog ────────────────────────────────────────────────────────────

var catalog = []nodeTypeDef{
	{Name: "Input", Color: "#22c55e", Icon: "\u25B6", Inputs: nil, Outputs: []string{"out"}},
	{Name: "Output", Color: "#ef4444", Icon: "\u25CF", Inputs: []string{"in"}, Outputs: nil},
	{Name: "Math", Color: "#3b82f6", Icon: "\u00B1", Inputs: []string{"a", "b"}, Outputs: []string{"result"}},
	{Name: "Filter", Color: "#a855f7", Icon: "\u25C7", Inputs: []string{"data"}, Outputs: []string{"pass", "fail"}},
	{Name: "Transform", Color: "#f59e0b", Icon: "\u21C4", Inputs: []string{"in"}, Outputs: []string{"out"}},
	{Name: "Merge", Color: "#06b6d4", Icon: "\u2A01", Inputs: []string{"a", "b", "c"}, Outputs: []string{"merged"}},
}

func catalogByName(name string) nodeTypeDef {
	for _, c := range catalog {
		if c.Name == name {
			return c
		}
	}
	return catalog[0]
}

func ntHeight(nt nodeTypeDef) float64 {
	n := len(nt.Inputs)
	if len(nt.Outputs) > n {
		n = len(nt.Outputs)
	}
	if n == 0 {
		n = 1
	}
	return float64(headerH + portPadY*2 + n*portRowH)
}

func findNodeIdx(nodes []nodeData, id string) int {
	for i, n := range nodes {
		if n.ID == id {
			return i
		}
	}
	return -1
}

// ── Port world position ────────────────────────────────────────────────

func portWorldXY(nd nodeData, isOutput bool, idx int) (float64, float64) {
	var x float64
	if isOutput {
		x = nd.X + nodeW
	} else {
		x = nd.X
	}
	y := nd.Y + float64(headerH) + float64(portPadY) + float64(idx)*float64(portRowH) + float64(portRowH)/2
	return x, y
}

// ── SVG helpers ────────────────────────────────────────────────────────

func writeBezier(sb *strings.Builder, x1, y1, x2, y2 float64, color string) {
	dx := math.Abs(x2-x1) * 0.5
	if dx < 50 {
		dx = 50
	}
	fmt.Fprintf(sb,
		`<path d="M%.1f %.1fC%.1f %.1f %.1f %.1f %.1f %.1f" fill="none" stroke="%s" stroke-width="2.5" stroke-linecap="round"/>`,
		x1, y1, x1+dx, y1, x2-dx, y2, x2, y2, color)
}

func buildSVGContent(nodes []nodeData, conns []connData, tempX1, tempY1, tempX2, tempY2 float64, tempActive bool) string {
	var sb strings.Builder
	for _, c := range conns {
		fi := findNodeIdx(nodes, c.FromNode)
		ti := findNodeIdx(nodes, c.ToNode)
		if fi < 0 || ti < 0 {
			continue
		}
		x1, y1 := portWorldXY(nodes[fi], true, c.FromPort)
		x2, y2 := portWorldXY(nodes[ti], false, c.ToPort)
		writeBezier(&sb, x1, y1, x2, y2, "#94a3b8")
	}
	if tempActive {
		writeBezier(&sb, tempX1, tempY1, tempX2, tempY2, "#f59e0b")
	}
	return sb.String()
}

// ── ID generation ──────────────────────────────────────────────────────

var nextID = 100

func genID() string {
	nextID++
	return fmt.Sprintf("n%d", nextID)
}

// ── Pure logic functions ───────────────────────────────────────────────

// deleteNode removes the node with the given ID and any connections
// touching it. Returns the updated slices.
func deleteNode(nodes []nodeData, conns []connData, id string) ([]nodeData, []connData) {
	idx := findNodeIdx(nodes, id)
	if idx < 0 {
		jsutil.LogWarn("deleteNode: node %s not found", id)
		return nodes, conns
	}
	removed := nodes[idx]
	nodes = append(nodes[:idx], nodes[idx+1:]...)
	var kept []connData
	dropped := 0
	for _, c := range conns {
		if c.FromNode != id && c.ToNode != id {
			kept = append(kept, c)
		} else {
			dropped++
		}
	}
	jsutil.LogInfo("deleteNode: removed %s (%s), pruned %d connections", removed.ID, removed.Type, dropped)
	return nodes, kept
}

// addConnection validates and appends a new connection. It ensures the
// connection goes from an output port to an input port on different nodes.
// Returns the updated slice and whether a connection was added.
func addConnection(conns []connData, fromNode string, fromPort int, toNode string, toPort int, isFromOutput bool) ([]connData, bool) {
	if fromNode == toNode {
		jsutil.LogWarn("addConnection: rejected self-loop on node %s", fromNode)
		return conns, false
	}
	if isFromOutput {
		conns = append(conns, connData{
			FromNode: fromNode, FromPort: fromPort,
			ToNode: toNode, ToPort: toPort,
		})
		jsutil.LogInfo("addConnection: %s:%d → %s:%d", fromNode, fromPort, toNode, toPort)
		return conns, true
	}
	// isFromOutput == false means the drag started from an input,
	// so the target must be an output: swap roles.
	conns = append(conns, connData{
		FromNode: toNode, FromPort: toPort,
		ToNode: fromNode, ToPort: fromPort,
	})
	jsutil.LogInfo("addConnection: %s:%d → %s:%d (swapped)", toNode, toPort, fromNode, fromPort)
	return conns, true
}

// centerGraphBounds computes the bounding box of all nodes.
// Returns (minX, minY, maxX, maxY). Assumes len(nodes) > 0.
func centerGraphBounds(nodes []nodeData) (float64, float64, float64, float64) {
	minX, minY := nodes[0].X, nodes[0].Y
	maxX := nodes[0].X + nodeW
	maxY := nodes[0].Y + ntHeight(catalogByName(nodes[0].Type))
	for _, nd := range nodes[1:] {
		nt := catalogByName(nd.Type)
		h := ntHeight(nt)
		if nd.X < minX {
			minX = nd.X
		}
		if nd.Y < minY {
			minY = nd.Y
		}
		if nd.X+nodeW > maxX {
			maxX = nd.X + nodeW
		}
		if nd.Y+h > maxY {
			maxY = nd.Y + h
		}
	}
	return minX, minY, maxX, maxY
}

// calcCenterView computes panX, panY, and zoom to center the given
// bounding box within a canvas of size canvasW x canvasH.
func calcCenterView(boundsMinX, boundsMinY, boundsMaxX, boundsMaxY, canvasW, canvasH float64) (float64, float64, float64) {
	gw := boundsMaxX - boundsMinX
	gh := boundsMaxY - boundsMinY
	pad := 80.0
	sx := canvasW / (gw + pad*2)
	sy := canvasH / (gh + pad*2)
	z := sx
	if sy < z {
		z = sy
	}
	if z > 1.5 {
		z = 1.5
	}
	if z < minZoom {
		z = minZoom
	}
	cx := boundsMinX + gw/2
	cy := boundsMinY + gh/2
	px := canvasW/2 - cx*z
	py := canvasH/2 - cy*z
	return px, py, z
}

// addNode creates a new node of the given type at (x, y) and appends it.
func addNode(nodes []nodeData, typeName string, x, y float64) []nodeData {
	id := genID()
	jsutil.LogInfo("addNode: %s (type=%s) at (%.0f, %.0f)", id, typeName, x, y)
	return append(nodes, nodeData{
		ID:   id,
		Type: typeName,
		X:    x,
		Y:    y,
	})
}
