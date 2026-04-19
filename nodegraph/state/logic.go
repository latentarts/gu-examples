//go:build js && wasm

package state

import (
	"fmt"
	"math"
	"strings"

	"github.com/latentart/gu/jsutil"
)

const (
	NodeW    = 180
	HeaderH  = 36
	PortRowH = 28
	PortPadY = 12
	PortR    = 6
	MinZoom  = 0.15
	MaxZoom  = 3.0
)

type NodeData struct {
	ID   string
	Type string
	X, Y float64
}

type ConnData struct {
	FromNode string
	FromPort int
	ToNode   string
	ToPort   int
}

type NodeTypeDef struct {
	Name    string
	Color   string
	Icon    string
	Inputs  []string
	Outputs []string
}

var Catalog = []NodeTypeDef{
	{Name: "Input", Color: "#22c55e", Icon: "\u25B6", Inputs: nil, Outputs: []string{"out"}},
	{Name: "Output", Color: "#ef4444", Icon: "\u25CF", Inputs: []string{"in"}, Outputs: nil},
	{Name: "Math", Color: "#3b82f6", Icon: "\u00B1", Inputs: []string{"a", "b"}, Outputs: []string{"result"}},
	{Name: "Filter", Color: "#a855f7", Icon: "\u25C7", Inputs: []string{"data"}, Outputs: []string{"pass", "fail"}},
	{Name: "Transform", Color: "#f59e0b", Icon: "\u21C4", Inputs: []string{"in"}, Outputs: []string{"out"}},
	{Name: "Merge", Color: "#06b6d4", Icon: "\u2A01", Inputs: []string{"a", "b", "c"}, Outputs: []string{"merged"}},
}

func CatalogByName(name string) NodeTypeDef {
	for _, c := range Catalog {
		if c.Name == name {
			return c
		}
	}
	return Catalog[0]
}

func NtHeight(nt NodeTypeDef) float64 {
	n := len(nt.Inputs)
	if len(nt.Outputs) > n {
		n = len(nt.Outputs)
	}
	if n == 0 {
		n = 1
	}
	return float64(HeaderH + PortPadY*2 + n*PortRowH)
}

func FindNodeIdx(nodes []NodeData, id string) int {
	for i, n := range nodes {
		if n.ID == id {
			return i
		}
	}
	return -1
}

func PortWorldXY(nd NodeData, isOutput bool, idx int) (float64, float64) {
	var x float64
	if isOutput {
		x = nd.X + NodeW
	} else {
		x = nd.X
	}
	y := nd.Y + float64(HeaderH) + float64(PortPadY) + float64(idx)*float64(PortRowH) + float64(PortRowH)/2
	return x, y
}

func writeBezier(sb *strings.Builder, x1, y1, x2, y2 float64, color string) {
	dx := math.Abs(x2-x1) * 0.5
	if dx < 50 {
		dx = 50
	}
	fmt.Fprintf(sb, `<path d="M%.1f %.1fC%.1f %.1f %.1f %.1f %.1f %.1f" fill="none" stroke="%s" stroke-width="2.5" stroke-linecap="round"/>`, x1, y1, x1+dx, y1, x2-dx, y2, x2, y2, color)
}

func BuildSVGContent(nodes []NodeData, conns []ConnData, tempX1, tempY1, tempX2, tempY2 float64, tempActive bool) string {
	var sb strings.Builder
	for _, c := range conns {
		fi := FindNodeIdx(nodes, c.FromNode)
		ti := FindNodeIdx(nodes, c.ToNode)
		if fi < 0 || ti < 0 {
			continue
		}
		x1, y1 := PortWorldXY(nodes[fi], true, c.FromPort)
		x2, y2 := PortWorldXY(nodes[ti], false, c.ToPort)
		writeBezier(&sb, x1, y1, x2, y2, "#94a3b8")
	}
	if tempActive {
		writeBezier(&sb, tempX1, tempY1, tempX2, tempY2, "#f59e0b")
	}
	return sb.String()
}

var nextID = 100

func genID() string {
	nextID++
	return fmt.Sprintf("n%d", nextID)
}

func DeleteNode(nodes []NodeData, conns []ConnData, id string) ([]NodeData, []ConnData) {
	idx := FindNodeIdx(nodes, id)
	if idx < 0 {
		jsutil.LogWarn("deleteNode: node %s not found", id)
		return nodes, conns
	}
	removed := nodes[idx]
	nodes = append(nodes[:idx], nodes[idx+1:]...)
	var kept []ConnData
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

func AddConnection(conns []ConnData, fromNode string, fromPort int, toNode string, toPort int, isFromOutput bool) ([]ConnData, bool) {
	if fromNode == toNode {
		jsutil.LogWarn("addConnection: rejected self-loop on node %s", fromNode)
		return conns, false
	}
	if isFromOutput {
		conns = append(conns, ConnData{FromNode: fromNode, FromPort: fromPort, ToNode: toNode, ToPort: toPort})
		jsutil.LogInfo("addConnection: %s:%d → %s:%d", fromNode, fromPort, toNode, toPort)
		return conns, true
	}
	conns = append(conns, ConnData{FromNode: toNode, FromPort: toPort, ToNode: fromNode, ToPort: fromPort})
	jsutil.LogInfo("addConnection: %s:%d → %s:%d (swapped)", toNode, toPort, fromNode, fromPort)
	return conns, true
}

func CenterGraphBounds(nodes []NodeData) (float64, float64, float64, float64) {
	minX, minY := nodes[0].X, nodes[0].Y
	maxX := nodes[0].X + NodeW
	maxY := nodes[0].Y + NtHeight(CatalogByName(nodes[0].Type))
	for _, nd := range nodes[1:] {
		nt := CatalogByName(nd.Type)
		h := NtHeight(nt)
		if nd.X < minX {
			minX = nd.X
		}
		if nd.Y < minY {
			minY = nd.Y
		}
		if nd.X+NodeW > maxX {
			maxX = nd.X + NodeW
		}
		if nd.Y+h > maxY {
			maxY = nd.Y + h
		}
	}
	return minX, minY, maxX, maxY
}

func CalcCenterView(boundsMinX, boundsMinY, boundsMaxX, boundsMaxY, canvasW, canvasH float64) (float64, float64, float64) {
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
	if z < MinZoom {
		z = MinZoom
	}
	cx := boundsMinX + gw/2
	cy := boundsMinY + gh/2
	px := canvasW/2 - cx*z
	py := canvasH/2 - cy*z
	return px, py, z
}

func AddNode(nodes []NodeData, typeName string, x, y float64) []NodeData {
	id := genID()
	jsutil.LogInfo("addNode: %s (type=%s) at (%.0f, %.0f)", id, typeName, x, y)
	return append(nodes, NodeData{ID: id, Type: typeName, X: x, Y: y})
}
