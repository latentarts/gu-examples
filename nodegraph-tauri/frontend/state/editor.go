package state

import (
	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/reactive"
)

type EditorState struct {
	Nodes []NodeData
	Conns []ConnData

	NodeVer    func() int
	SetNodeVer func(int)
	ConnVer    func() int
	SetConnVer func(int)

	PanX    func() float64
	SetPanX func(float64)
	PanY    func() float64
	SetPanY func(float64)
	Zoom    func() float64
	SetZoom func(float64)

	DrawerOpen    func() bool
	SetDrawerOpen func(bool)

	TempActive    func() bool
	SetTempActive func(bool)
	TempX1        func() float64
	SetTempX1     func(float64)
	TempY1        func() float64
	SetTempY1     func(float64)
	TempX2        func() float64
	SetTempX2     func(float64)
	TempY2        func() float64
	SetTempY2     func(float64)

	PlacingType    func() string
	SetPlacingType func(string)

	CanvasEl       dom.Element
	WorldEl        dom.Element
	SVGContainerEl dom.Element
}

func NewEditorState() *EditorState {
	nodeVer, setNodeVer := reactive.NewSignal(0)
	connVer, setConnVer := reactive.NewSignal(0)
	panX, setPanX := reactive.NewSignal(0.0)
	panY, setPanY := reactive.NewSignal(0.0)
	zoom, setZoom := reactive.NewSignal(1.0)
	drawerOpen, setDrawerOpen := reactive.NewSignal(true)
	tempActive, setTempActive := reactive.NewSignal(false)
	tempX1, setTempX1 := reactive.NewSignal(0.0)
	tempY1, setTempY1 := reactive.NewSignal(0.0)
	tempX2, setTempX2 := reactive.NewSignal(0.0)
	tempY2, setTempY2 := reactive.NewSignal(0.0)
	placingType, setPlacingType := reactive.NewSignal("")

	return &EditorState{
		Nodes: []NodeData{
			{ID: "n1", Type: "Input", X: 80, Y: 120},
			{ID: "n2", Type: "Transform", X: 380, Y: 100},
			{ID: "n3", Type: "Output", X: 680, Y: 140},
		},
		Conns: []ConnData{
			{FromNode: "n1", FromPort: 0, ToNode: "n2", ToPort: 0},
			{FromNode: "n2", FromPort: 0, ToNode: "n3", ToPort: 0},
		},
		NodeVer:       nodeVer,
		SetNodeVer:    setNodeVer,
		ConnVer:       connVer,
		SetConnVer:    setConnVer,
		PanX:          panX,
		SetPanX:       setPanX,
		PanY:          panY,
		SetPanY:       setPanY,
		Zoom:          zoom,
		SetZoom:       setZoom,
		DrawerOpen:    drawerOpen,
		SetDrawerOpen: setDrawerOpen,
		TempActive:    tempActive,
		SetTempActive: setTempActive,
		TempX1:        tempX1,
		SetTempX1:     setTempX1,
		TempY1:        tempY1,
		SetTempY1:     setTempY1,
		TempX2:        tempX2,
		SetTempX2:     setTempX2,
		TempY2:        tempY2,
		SetTempY2:     setTempY2,
		PlacingType:   placingType,
		SetPlacingType:setPlacingType,
	}
}

func (s *EditorState) BumpNodes() { s.SetNodeVer(s.NodeVer() + 1) }
func (s *EditorState) BumpConns() { s.SetConnVer(s.ConnVer() + 1) }
