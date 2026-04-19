//go:build js && wasm

package main

import (
	"strings"
	"testing"
)

func TestDeleteNode(t *testing.T) {
	nodes := []nodeData{
		{ID: "n1", Type: "Input", X: 0, Y: 0},
		{ID: "n2", Type: "Output", X: 100, Y: 0},
		{ID: "n3", Type: "Math", X: 200, Y: 0},
	}
	conns := []connData{
		{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0},
		{FromNode: "n1", FromPort: 0, ToNode: "n3", ToPort: 0},
		{FromNode: "n3", FromPort: 0, ToNode: "n2", ToPort: 0},
	}

	newNodes, newConns := deleteNode(nodes, conns, "n1")
	if len(newNodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(newNodes))
	}
	if newNodes[0].ID != "n2" || newNodes[1].ID != "n3" {
		t.Fatalf("unexpected node IDs: %v", newNodes)
	}
	// Only n3→n2 connection should remain
	if len(newConns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(newConns))
	}
	if newConns[0].FromNode != "n3" || newConns[0].ToNode != "n2" {
		t.Fatalf("unexpected remaining connection: %v", newConns[0])
	}
}

func TestDeleteNodeNonexistent(t *testing.T) {
	nodes := []nodeData{{ID: "n1", Type: "Input", X: 0, Y: 0}}
	conns := []connData{{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0}}

	newNodes, newConns := deleteNode(nodes, conns, "n99")
	if len(newNodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(newNodes))
	}
	if len(newConns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(newConns))
	}
}

func TestAddConnectionOutputToInput(t *testing.T) {
	var conns []connData
	newConns, ok := addConnection(conns, "n1", 0, "n2", 0, true)
	if !ok {
		t.Fatal("expected connection to be added")
	}
	if len(newConns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(newConns))
	}
	c := newConns[0]
	if c.FromNode != "n1" || c.ToNode != "n2" {
		t.Fatalf("unexpected connection: %v", c)
	}
}

func TestAddConnectionInputToOutput(t *testing.T) {
	var conns []connData
	// Drag started from input port (isFromOutput=false), dropped on output
	newConns, ok := addConnection(conns, "n2", 0, "n1", 0, false)
	if !ok {
		t.Fatal("expected connection to be added")
	}
	c := newConns[0]
	// Should be swapped: n1 (output) → n2 (input)
	if c.FromNode != "n1" || c.ToNode != "n2" {
		t.Fatalf("expected swapped connection, got: %v", c)
	}
}

func TestAddConnectionSameNode(t *testing.T) {
	var conns []connData
	newConns, ok := addConnection(conns, "n1", 0, "n1", 1, true)
	if ok {
		t.Fatal("expected same-node connection to be rejected")
	}
	if len(newConns) != 0 {
		t.Fatalf("expected 0 connections, got %d", len(newConns))
	}
}

func TestFindNodeIdx(t *testing.T) {
	nodes := []nodeData{
		{ID: "a"}, {ID: "b"}, {ID: "c"},
	}
	if idx := findNodeIdx(nodes, "b"); idx != 1 {
		t.Fatalf("expected 1, got %d", idx)
	}
	if idx := findNodeIdx(nodes, "z"); idx != -1 {
		t.Fatalf("expected -1, got %d", idx)
	}
}

func TestPortWorldXY(t *testing.T) {
	nd := nodeData{ID: "n1", Type: "Input", X: 100, Y: 200}

	// Output port 0
	x, y := portWorldXY(nd, true, 0)
	if x != 100+nodeW {
		t.Fatalf("expected x=%d, got %.1f", 100+nodeW, x)
	}
	expectedY := 200 + float64(headerH) + float64(portPadY) + float64(portRowH)/2
	if y != expectedY {
		t.Fatalf("expected y=%.1f, got %.1f", expectedY, y)
	}

	// Input port 0
	x, _ = portWorldXY(nd, false, 0)
	if x != 100 {
		t.Fatalf("expected x=100, got %.1f", x)
	}
}

func TestBuildSVGContent(t *testing.T) {
	nodes := []nodeData{
		{ID: "n1", Type: "Input", X: 0, Y: 0},
		{ID: "n2", Type: "Output", X: 300, Y: 0},
	}
	conns := []connData{
		{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0},
	}

	svg := buildSVGContent(nodes, conns, 0, 0, 0, 0, false)
	if !strings.Contains(svg, "<path") {
		t.Fatal("expected SVG path in output")
	}
	if !strings.Contains(svg, `stroke="#94a3b8"`) {
		t.Fatal("expected connection color in output")
	}

	// With temp line
	svg = buildSVGContent(nodes, conns, 10, 20, 30, 40, true)
	if strings.Count(svg, "<path") != 2 {
		t.Fatalf("expected 2 paths, got %d", strings.Count(svg, "<path"))
	}
	if !strings.Contains(svg, `stroke="#f59e0b"`) {
		t.Fatal("expected temp connection color in output")
	}
}

func TestBuildSVGContentMissingNode(t *testing.T) {
	nodes := []nodeData{{ID: "n1", Type: "Input", X: 0, Y: 0}}
	conns := []connData{{FromNode: "n1", FromPort: 0, ToNode: "n99", ToPort: 0}}

	svg := buildSVGContent(nodes, conns, 0, 0, 0, 0, false)
	if strings.Contains(svg, "<path") {
		t.Fatal("expected no path for missing node")
	}
}

func TestCenterGraphBounds(t *testing.T) {
	nodes := []nodeData{
		{ID: "n1", Type: "Input", X: 50, Y: 100},
		{ID: "n2", Type: "Output", X: 400, Y: 300},
	}

	minX, minY, maxX, maxY := centerGraphBounds(nodes)
	if minX != 50 {
		t.Fatalf("expected minX=50, got %.1f", minX)
	}
	if minY != 100 {
		t.Fatalf("expected minY=100, got %.1f", minY)
	}
	if maxX != 400+nodeW {
		t.Fatalf("expected maxX=%d, got %.1f", 400+nodeW, maxX)
	}
	// maxY = 300 + ntHeight(Output)
	nt := catalogByName("Output")
	expectedMaxY := 300 + ntHeight(nt)
	if maxY != expectedMaxY {
		t.Fatalf("expected maxY=%.1f, got %.1f", expectedMaxY, maxY)
	}
}

func TestCalcCenterView(t *testing.T) {
	// Small graph in large canvas → zoom clamped to 1.5
	px, py, z := calcCenterView(0, 0, 100, 100, 2000, 2000)
	if z != 1.5 {
		t.Fatalf("expected zoom clamped to 1.5, got %.4f", z)
	}
	// Center should be roughly in the middle
	expectedPX := 2000.0/2 - 50*1.5
	expectedPY := 2000.0/2 - 50*1.5
	if px != expectedPX {
		t.Fatalf("expected panX=%.1f, got %.1f", expectedPX, px)
	}
	if py != expectedPY {
		t.Fatalf("expected panY=%.1f, got %.1f", expectedPY, py)
	}

	// Large graph in small canvas → zoom < 1
	_, _, z = calcCenterView(0, 0, 5000, 5000, 800, 600)
	if z >= 1.0 {
		t.Fatalf("expected zoom < 1, got %.4f", z)
	}
	if z < minZoom {
		t.Fatalf("expected zoom >= minZoom, got %.4f", z)
	}
}

func TestCalcCenterViewMinZoom(t *testing.T) {
	// Extremely large graph → zoom clamped to minZoom
	_, _, z := calcCenterView(0, 0, 100000, 100000, 100, 100)
	if z != minZoom {
		t.Fatalf("expected zoom clamped to minZoom (%.2f), got %.4f", minZoom, z)
	}
}

func TestAddNode(t *testing.T) {
	nodes := []nodeData{{ID: "n1", Type: "Input", X: 0, Y: 0}}
	oldNextID := nextID

	newNodes := addNode(nodes, "Math", 100, 200)
	if len(newNodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(newNodes))
	}
	added := newNodes[1]
	if added.Type != "Math" {
		t.Fatalf("expected type Math, got %s", added.Type)
	}
	if added.X != 100 || added.Y != 200 {
		t.Fatalf("expected position (100, 200), got (%.1f, %.1f)", added.X, added.Y)
	}
	if added.ID == "" {
		t.Fatal("expected generated ID")
	}
	if nextID <= oldNextID {
		t.Fatal("expected nextID to have incremented")
	}
}

func TestNtHeight(t *testing.T) {
	tests := []struct {
		name     string
		nt       nodeTypeDef
		expected float64
	}{
		{
			name:     "no ports",
			nt:       nodeTypeDef{Name: "Empty"},
			expected: float64(headerH + portPadY*2 + 1*portRowH),
		},
		{
			name:     "one input",
			nt:       nodeTypeDef{Name: "X", Inputs: []string{"a"}},
			expected: float64(headerH + portPadY*2 + 1*portRowH),
		},
		{
			name:     "three inputs two outputs",
			nt:       nodeTypeDef{Name: "X", Inputs: []string{"a", "b", "c"}, Outputs: []string{"x", "y"}},
			expected: float64(headerH + portPadY*2 + 3*portRowH),
		},
		{
			name:     "two inputs three outputs",
			nt:       nodeTypeDef{Name: "X", Inputs: []string{"a", "b"}, Outputs: []string{"x", "y", "z"}},
			expected: float64(headerH + portPadY*2 + 3*portRowH),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ntHeight(tt.nt)
			if got != tt.expected {
				t.Fatalf("expected %.1f, got %.1f", tt.expected, got)
			}
		})
	}
}
