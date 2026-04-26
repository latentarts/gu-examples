package state

import (
	"strings"
	"testing"
)

func TestDeleteNode(t *testing.T) {
	nodes := []NodeData{
		{ID: "n1", Type: "Input", X: 0, Y: 0},
		{ID: "n2", Type: "Output", X: 100, Y: 0},
		{ID: "n3", Type: "Math", X: 200, Y: 0},
	}
	conns := []ConnData{
		{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0},
		{FromNode: "n1", FromPort: 0, ToNode: "n3", ToPort: 0},
		{FromNode: "n3", FromPort: 0, ToNode: "n2", ToPort: 0},
	}
	newNodes, newConns := DeleteNode(nodes, conns, "n1")
	if len(newNodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(newNodes))
	}
	if newNodes[0].ID != "n2" || newNodes[1].ID != "n3" {
		t.Fatalf("unexpected node IDs: %v", newNodes)
	}
	if len(newConns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(newConns))
	}
	if newConns[0].FromNode != "n3" || newConns[0].ToNode != "n2" {
		t.Fatalf("unexpected remaining connection: %v", newConns[0])
	}
}

func TestDeleteNodeNonexistent(t *testing.T) {
	nodes := []NodeData{{ID: "n1", Type: "Input", X: 0, Y: 0}}
	conns := []ConnData{{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0}}
	newNodes, newConns := DeleteNode(nodes, conns, "n99")
	if len(newNodes) != 1 || len(newConns) != 1 {
		t.Fatal("expected unchanged slices")
	}
}

func TestAddConnectionOutputToInput(t *testing.T) {
	var conns []ConnData
	newConns, ok := AddConnection(conns, "n1", 0, "n2", 0, true)
	if !ok || len(newConns) != 1 {
		t.Fatal("expected connection to be added")
	}
}

func TestAddConnectionInputToOutput(t *testing.T) {
	var conns []ConnData
	newConns, ok := AddConnection(conns, "n2", 0, "n1", 0, false)
	if !ok {
		t.Fatal("expected connection to be added")
	}
	c := newConns[0]
	if c.FromNode != "n1" || c.ToNode != "n2" {
		t.Fatalf("expected swapped connection, got: %v", c)
	}
}

func TestAddConnectionSameNode(t *testing.T) {
	var conns []ConnData
	newConns, ok := AddConnection(conns, "n1", 0, "n1", 1, true)
	if ok || len(newConns) != 0 {
		t.Fatal("expected same-node connection to be rejected")
	}
}

func TestFindNodeIdx(t *testing.T) {
	nodes := []NodeData{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	if idx := FindNodeIdx(nodes, "b"); idx != 1 {
		t.Fatalf("expected 1, got %d", idx)
	}
	if idx := FindNodeIdx(nodes, "z"); idx != -1 {
		t.Fatalf("expected -1, got %d", idx)
	}
}

func TestPortWorldXY(t *testing.T) {
	nd := NodeData{ID: "n1", Type: "Input", X: 100, Y: 200}
	x, y := PortWorldXY(nd, true, 0)
	if x != 100+NodeW {
		t.Fatalf("expected x=%d, got %.1f", 100+NodeW, x)
	}
	expectedY := 200 + float64(HeaderH) + float64(PortPadY) + float64(PortRowH)/2
	if y != expectedY {
		t.Fatalf("expected y=%.1f, got %.1f", expectedY, y)
	}
	x, _ = PortWorldXY(nd, false, 0)
	if x != 100 {
		t.Fatalf("expected x=100, got %.1f", x)
	}
}

func TestBuildSVGContent(t *testing.T) {
	nodes := []NodeData{
		{ID: "n1", Type: "Input", X: 0, Y: 0},
		{ID: "n2", Type: "Output", X: 300, Y: 0},
	}
	conns := []ConnData{{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0}}
	svg := BuildSVGContent(nodes, conns, 0, 0, 0, 0, false)
	if !strings.Contains(svg, "<path") || !strings.Contains(svg, `stroke="#94a3b8"`) {
		t.Fatal("expected connection path in output")
	}
	svg = BuildSVGContent(nodes, conns, 10, 20, 30, 40, true)
	if strings.Count(svg, "<path") != 2 || !strings.Contains(svg, `stroke="#f59e0b"`) {
		t.Fatal("expected temp path in output")
	}
}

func TestBuildSVGContentMissingNode(t *testing.T) {
	nodes := []NodeData{{ID: "n1", Type: "Input", X: 0, Y: 0}}
	conns := []ConnData{{FromNode: "n1", FromPort: 0, ToNode: "n99", ToPort: 0}}
	if svg := BuildSVGContent(nodes, conns, 0, 0, 0, 0, false); strings.Contains(svg, "<path") {
		t.Fatal("expected no path for missing node")
	}
}

func TestCenterGraphBounds(t *testing.T) {
	nodes := []NodeData{
		{ID: "n1", Type: "Input", X: 50, Y: 100},
		{ID: "n2", Type: "Output", X: 400, Y: 300},
	}
	minX, minY, maxX, maxY := CenterGraphBounds(nodes)
	if minX != 50 || minY != 100 {
		t.Fatal("unexpected minimum bounds")
	}
	if maxX != 400+NodeW {
		t.Fatalf("expected maxX=%d, got %.1f", 400+NodeW, maxX)
	}
	expectedMaxY := 300 + NtHeight(CatalogByName("Output"))
	if maxY != expectedMaxY {
		t.Fatalf("expected maxY=%.1f, got %.1f", expectedMaxY, maxY)
	}
}

func TestCalcCenterView(t *testing.T) {
	px, py, z := CalcCenterView(0, 0, 100, 100, 2000, 2000)
	if z != 1.5 {
		t.Fatalf("expected zoom clamped to 1.5, got %.4f", z)
	}
	if px != 2000.0/2-50*1.5 || py != 2000.0/2-50*1.5 {
		t.Fatal("unexpected pan values")
	}
	_, _, z = CalcCenterView(0, 0, 5000, 5000, 800, 600)
	if z >= 1.0 || z < MinZoom {
		t.Fatalf("unexpected zoom %.4f", z)
	}
}

func TestCalcCenterViewMinZoom(t *testing.T) {
	_, _, z := CalcCenterView(0, 0, 100000, 100000, 100, 100)
	if z != MinZoom {
		t.Fatalf("expected min zoom %.2f, got %.4f", MinZoom, z)
	}
}

func TestAddNode(t *testing.T) {
	nodes := []NodeData{{ID: "n1", Type: "Input", X: 0, Y: 0}}
	oldNextID := nextID
	newNodes := AddNode(nodes, "Math", 100, 200)
	if len(newNodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(newNodes))
	}
	added := newNodes[1]
	if added.Type != "Math" || added.X != 100 || added.Y != 200 || added.ID == "" || nextID <= oldNextID {
		t.Fatal("expected node to be appended with generated id")
	}
}

func TestNtHeight(t *testing.T) {
	tests := []struct {
		 nt       NodeTypeDef
		 expected float64
	}{
		{nt: NodeTypeDef{Name: "Empty"}, expected: float64(HeaderH + PortPadY*2 + 1*PortRowH)},
		{nt: NodeTypeDef{Name: "X", Inputs: []string{"a"}}, expected: float64(HeaderH + PortPadY*2 + 1*PortRowH)},
		{nt: NodeTypeDef{Name: "X", Inputs: []string{"a", "b", "c"}, Outputs: []string{"x", "y"}}, expected: float64(HeaderH + PortPadY*2 + 3*PortRowH)},
	}
	for _, tt := range tests {
		if got := NtHeight(tt.nt); got != tt.expected {
			t.Fatalf("expected %.1f, got %.1f", tt.expected, got)
		}
	}
}
