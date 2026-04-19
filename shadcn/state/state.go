package state

import (
	"time"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/reactive"
)

type DrawerState struct {
	Open       func() bool
	SetOpen    func(bool)
	OffsetY    func() float64
	SetOffsetY func(float64)
	Dragging   func() bool
	SetDragging func(bool)
	Goal       func() int
	SetGoal    func(int)
}

func NewDrawerState() *DrawerState {
	open, setOpen := reactive.NewSignal(false)
	offsetY, setOffsetY := reactive.NewSignal(0.0)
	dragging, setDragging := reactive.NewSignal(false)
	goal, setGoal := reactive.NewSignal(350)
	return &DrawerState{open, setOpen, offsetY, setOffsetY, dragging, setDragging, goal, setGoal}
}

type DatePickerState struct {
	SelYear     func() int
	SetSelYear  func(int)
	SelMonth    func() int
	SetSelMonth func(int)
	SelDay      func() int
	SetSelDay   func(int)
	ViewYear    func() int
	SetViewYear func(int)
	ViewMonth   func() int
	SetViewMonth func(int)
	Open        func() bool
	SetOpen     func(bool)
}

func NewDatePickerState(now time.Time) *DatePickerState {
	selYear, setSelYear := reactive.NewSignal(now.Year())
	selMonth, setSelMonth := reactive.NewSignal(int(now.Month()))
	selDay, setSelDay := reactive.NewSignal(now.Day())
	viewYear, setViewYear := reactive.NewSignal(now.Year())
	viewMonth, setViewMonth := reactive.NewSignal(int(now.Month()))
	open, setOpen := reactive.NewSignal(false)
	return &DatePickerState{
		selYear, setSelYear,
		selMonth, setSelMonth,
		selDay, setSelDay,
		viewYear, setViewYear,
		viewMonth, setViewMonth,
		open, setOpen,
	}
}

type CarouselState struct {
	Current    func() int
	SetCurrent func(int)
}

func NewCarouselState() *CarouselState {
	current, setCurrent := reactive.NewSignal(0)
	return &CarouselState{current, setCurrent}
}

type ButtonGroupState struct {
	Selected    func() int
	SetSelected func(int)
}

func NewButtonGroupState() *ButtonGroupState {
	selected, setSelected := reactive.NewSignal(0)
	return &ButtonGroupState{selected, setSelected}
}

type ResizableState struct {
	SplitPct    func() float64
	SetSplitPct func(float64)
	ContainerEl dom.Element
}

func NewResizableState() *ResizableState {
	splitPct, setSplitPct := reactive.NewSignal(50.0)
	return &ResizableState{
		SplitPct:    splitPct,
		SetSplitPct: setSplitPct,
	}
}
