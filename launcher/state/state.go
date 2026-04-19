//go:build js && wasm

package state

import (
	"github.com/latentart/gu/reactive"
)

// LauncherState manages the reactive state of the launcher application.
type LauncherState struct {
	Selected    func() string
	SetSelected func(string)

	SidebarOpen    func() bool
	SetSidebarOpen func(bool)
}

// NewLauncherState initializes a new launcher state.
func NewLauncherState() *LauncherState {
	selected, setSelected := reactive.NewSignal("")
	sidebarOpen, setSidebarOpen := reactive.NewSignal(true)

	return &LauncherState{
		Selected:       selected,
		SetSelected:    setSelected,
		SidebarOpen:    sidebarOpen,
		SetSidebarOpen: setSidebarOpen,
	}
}