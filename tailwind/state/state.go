package state

import "github.com/latentart/gu/reactive"

type ShowcaseState struct {
	IsDark    func() bool
	SetIsDark func(bool)
}

func NewShowcaseState() *ShowcaseState {
	isDark, setIsDark := reactive.NewSignal(false)
	return &ShowcaseState{
		IsDark:    isDark,
		SetIsDark: setIsDark,
	}
}
