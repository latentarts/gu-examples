//go:build js && wasm

package components

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/latentart/gu/dom"
	"github.com/latentart/gu/el"
	"github.com/latentarts/gu-examples/webgpu/state"
)

func Screen(styles el.Node, s *state.AppState) el.Node {
	return el.Div(
		styles,
		el.Class("min-h-screen bg-gray-950 text-gray-100"),
		el.Div(
			el.Class("max-w-4xl mx-auto px-6 py-8"),
			el.Div(
				el.Class("flex items-center justify-between mb-6"),
				el.Div(
					el.H1(el.Class("text-2xl font-bold text-brand"), el.Text("WebGPU Cube")),
					el.P(el.Class("text-xs text-gray-500 mt-1"), el.Text("Rotating 3D cube rendered with WebGPU, logic fully in Go WASM")),
				),
				el.Show(func() bool { return s.Supported() && s.Running() },
					el.Span(el.Class("text-xs px-3 py-1 bg-green-500/20 text-green-400 rounded-full"), el.Text("WebGPU Active")),
				),
			),
			el.Show(func() bool { return s.ErrMsg() != "" },
				el.Div(el.Class("bg-red-900/50 text-red-300 rounded-lg px-4 py-3 mb-6 text-sm"), el.DynText(s.ErrMsg)),
			),
			el.Show(func() bool { return !s.Supported() },
				el.Div(el.Class("bg-amber-900/50 text-amber-300 rounded-lg px-4 py-3 mb-6 text-sm"), el.Text("WebGPU not supported")),
			),
			el.Div(
				el.Class("rounded-xl overflow-hidden border border-gray-800 mb-6 relative"),
				el.Show(func() bool { return s.Running() },
					el.Div(
						el.Class("absolute top-4 left-4 bg-gray-900/80 backdrop-blur px-2 py-1 rounded border border-gray-700 text-xs font-mono text-gray-300 z-10"),
						el.DynText(func() string {
							return fmt.Sprintf("%d fps", s.FPS())
						}),
					),
				),
				el.Tag("canvas",
					el.Attr("width", "800"), el.Attr("height", "500"),
					el.Class("w-full bg-gray-900"),
					el.OnMount(func(canvas dom.Element) {
						go func() {
							if err := s.Renderer.Init(canvas.Value); err != nil {
								s.SetSupported(false)
								s.SetErrMsg(err.Error())
								return
							}
							s.SetRunning(true)
							angle := 0.0
							var lastTime float64
							frameCount := 0

							var renderFrame js.Func
							renderFrame = js.FuncOf(func(this js.Value, args []js.Value) any {
								now := js.Global().Get("performance").Call("now").Float()
								if lastTime > 0 {
									dt := (now - lastTime) / 1000.0
									angle += dt * s.Speed() * math.Pi
									frameCount++
									if frameCount >= 30 {
										s.SetFPS(int(math.Round(1.0 / dt)))
										frameCount = 0
									}
								}
								lastTime = now

								s.Renderer.Render(float32(angle), 800, 500)
								js.Global().Call("requestAnimationFrame", renderFrame)
								return nil
							})
							js.Global().Call("requestAnimationFrame", renderFrame)
						}()
					}),
				),
			),
			el.Show(func() bool { return s.Running() },
				el.Div(
					el.Class("space-y-6"),
					el.Div(el.Class("bg-gray-900 rounded-xl p-4 border border-gray-800"),
						el.Div(el.Class("flex items-center justify-between mb-2"),
							el.Span(el.Class("text-sm font-medium"), el.Text("Rotation Speed")),
							el.Span(el.Class("text-sm text-gray-400"), el.DynText(func() string { return fmt.Sprintf("%.1fx", s.Speed()) })),
						),
						el.Input(el.Type("range"), el.Attr("min", "0"), el.Attr("max", "5"), el.Attr("step", "0.1"), el.Value("1"),
							el.Class("w-full accent-brand"),
							el.OnInput(func(e dom.Event) {
								var f float64
								fmt.Sscanf(e.TargetValue(), "%f", &f)
								s.SetSpeed(f)
							}),
						),
					),
					el.Div(el.Class("bg-gray-900 rounded-xl p-4 border border-gray-800"),
						el.Span(el.Class("text-sm font-medium block mb-3"), el.Text("Color Presets")),
						colorButtonGrid(s),
					),
				),
			),
		),
	)
}

func colorButtonGrid(s *state.AppState) el.Node {
	args := []any{el.Class("flex flex-wrap gap-2")}
	for _, p := range state.Presets {
		p := p
		bg := fmt.Sprintf("rgb(%d,%d,%d)", int(p.R*255), int(p.G*255), int(p.B*255))
		args = append(args, el.Button(
			el.Class("px-3 py-1.5 rounded-lg text-xs font-medium bg-gray-800 hover:bg-gray-700 transition-colors flex items-center gap-2"),
			el.Span(el.Class("w-3 h-3 rounded-full"), el.Style("background-color", bg)),
			el.Text(p.Name),
			el.OnClick(func(e dom.Event) {
				s.Renderer.SetColor(p.Face, float32(p.R), float32(p.G), float32(p.B))
			}),
		))
	}
	return el.Div(args...)
}
