//go:build js && wasm

package app

import (
	"testing"

	"github.com/latentart/gu/el"
	"github.com/latentart/gu/reactive"
	"github.com/latentarts/gu-examples/tailwind/components"
	"github.com/latentarts/gu-examples/tailwind/state"
)

func TestAppAndCardsRender(t *testing.T) {
	reactive.CreateRoot(func(dispose func()) {
		defer dispose()
		s := state.NewShowcaseState()

		nodes := []struct {
			name string
			node any
		}{
			{name: "App", node: App(el.Tag("style"))},
			{name: "Showcase", node: components.Showcase(el.Tag("style"), s)},
			{name: "HeroCard", node: components.HeroCard()},
			{name: "ProfileCard", node: components.ProfileCard()},
			{name: "PricingCard", node: components.PricingCard()},
			{name: "StatsCard", node: components.StatsCard()},
			{name: "TestimonialCard", node: components.TestimonialCard()},
			{name: "CTACard", node: components.CTACard()},
		}

		for _, tc := range nodes {
			if tc.node == nil {
				t.Fatalf("%s returned nil", tc.name)
			}
		}
	})
}
