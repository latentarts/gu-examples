//go:build js && wasm

package state

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/latentart/gu/jsutil"
	"github.com/latentart/gu/reactive"
)

type Mat4 [16]float32

func Mat4Identity() Mat4 {
	return Mat4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1}
}

func Mat4Multiply(a, b Mat4) Mat4 {
	var out Mat4
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+r] * b[c*4+k]
			}
			out[c*4+r] = s
		}
	}
	return out
}

func Mat4Perspective(fov, aspect, near, far float32) Mat4 {
	f := float32(1.0 / math.Tan(float64(fov)/2.0))
	nf := 1.0 / (near - far)
	var out Mat4
	out[0] = f / aspect
	out[5] = f
	out[10] = far * nf
	out[11] = -1
	out[14] = far * near * nf
	return out
}

func Mat4LookAt(eye, center, up [3]float32) Mat4 {
	sub := func(a, b [3]float32) [3]float32 { return [3]float32{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
	dot := func(a, b [3]float32) float32 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
	cross := func(a, b [3]float32) [3]float32 {
		return [3]float32{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
	}
	normalize := func(v [3]float32) [3]float32 {
		l := float32(math.Sqrt(float64(dot(v, v))))
		return [3]float32{v[0] / l, v[1] / l, v[2] / l}
	}

	z := normalize(sub(eye, center))
	x := normalize(cross(up, z))
	y := cross(z, x)

	var out Mat4
	out[0], out[1], out[2] = x[0], y[0], z[0]
	out[4], out[5], out[6] = x[1], y[1], z[1]
	out[8], out[9], out[10] = x[2], y[2], z[2]
	out[12] = -dot(x, eye)
	out[13] = -dot(y, eye)
	out[14] = -dot(z, eye)
	out[15] = 1
	return out
}

func Mat4RotateY(a float32) Mat4 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	out := Mat4Identity()
	out[0], out[2], out[8], out[10] = c, -s, s, c
	return out
}

func Mat4RotateX(a float32) Mat4 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	out := Mat4Identity()
	out[5], out[6], out[9], out[10] = c, s, -s, c
	return out
}

const shaderCode = `
    struct Uniforms { mvp: mat4x4f };
    @group(0) @binding(0) var<uniform> u: Uniforms;
    struct VSOut { @builtin(position) pos: vec4f, @location(0) color: vec3f };
    @vertex fn vs(@location(0) pos: vec3f, @location(1) color: vec3f) -> VSOut {
        var o: VSOut;
        o.pos = u.mvp * vec4f(pos, 1.0);
        o.color = color;
        return o;
    }
    @fragment fn fs(@location(0) color: vec3f) -> @location(0) vec4f {
        return vec4f(color, 1.0);
    }
`

type WebGPUState struct {
	device           js.Value
	context          js.Value
	pipeline         js.Value
	vertexBuffer     js.Value
	indexBuffer      js.Value
	uniformBuffer    js.Value
	uniformBindGroup js.Value
	depthTexture     js.Value
	numIndices       int
	vertexData       []float32
}

func (s *WebGPUState) Init(canvas js.Value) error {
	gpu := js.Global().Get("navigator").Get("gpu")
	if gpu.IsUndefined() {
		return fmt.Errorf("WebGPU not supported")
	}

	adapter, err := jsutil.Await(gpu.Call("requestAdapter"))
	if err != nil || adapter.IsNull() {
		return fmt.Errorf("no WebGPU adapter found")
	}

	device, err := jsutil.Await(adapter.Call("requestDevice"))
	if err != nil {
		return fmt.Errorf("failed to request device: %w", err)
	}
	s.device = device
	s.context = canvas.Call("getContext", "webgpu")
	format := gpu.Call("getPreferredCanvasFormat")
	s.context.Call("configure", map[string]any{
		"device":    device,
		"format":    format,
		"alphaMode": "premultiplied",
	})

	shaderModule := device.Call("createShaderModule", map[string]any{"code": shaderCode})
	s.pipeline = device.Call("createRenderPipeline", map[string]any{
		"layout": "auto",
		"vertex": map[string]any{
			"module":     shaderModule,
			"entryPoint": "vs",
			"buffers": []any{
				map[string]any{
					"arrayStride": 24,
					"attributes": []any{
						map[string]any{"shaderLocation": 0, "offset": 0, "format": "float32x3"},
						map[string]any{"shaderLocation": 1, "offset": 12, "format": "float32x3"},
					},
				},
			},
		},
		"fragment": map[string]any{
			"module":     shaderModule,
			"entryPoint": "fs",
			"targets":    []any{map[string]any{"format": format}},
		},
		"primitive":    map[string]any{"topology": "triangle-list", "cullMode": "back"},
		"depthStencil": map[string]any{"format": "depth24plus", "depthWriteEnabled": true, "depthCompare": "less"},
	})

	verts, indices := buildCubeData()
	s.vertexData = verts
	s.numIndices = len(indices)
	s.vertexBuffer = device.Call("createBuffer", map[string]any{
		"size":  len(verts) * 4,
		"usage": 0x0020 | 0x0008,
	})
	s.updateVertexBuffer()

	idxBuf := device.Call("createBuffer", map[string]any{
		"size":  len(indices) * 2,
		"usage": 0x0010 | 0x0008,
	})
	s.device.Get("queue").Call("writeBuffer", idxBuf, 0, s.toTypedArray(indices))
	s.indexBuffer = idxBuf

	s.uniformBuffer = device.Call("createBuffer", map[string]any{
		"size":  64,
		"usage": 0x0040 | 0x0008,
	})

	s.uniformBindGroup = device.Call("createBindGroup", map[string]any{
		"layout": s.pipeline.Call("getBindGroupLayout", 0),
		"entries": []any{
			map[string]any{"binding": 0, "resource": map[string]any{"buffer": s.uniformBuffer}},
		},
	})
	return nil
}

func (s *WebGPUState) updateVertexBuffer() {
	s.device.Get("queue").Call("writeBuffer", s.vertexBuffer, 0, s.toTypedArray(s.vertexData))
}

func (s *WebGPUState) toTypedArray(data any) js.Value {
	var uint8 js.Value
	switch v := data.(type) {
	case []float32:
		b := s.float32ToBytes(v)
		uint8 = js.Global().Get("Uint8Array").New(len(b))
		js.CopyBytesToJS(uint8, b)
		return js.Global().Get("Float32Array").New(uint8.Get("buffer"), uint8.Get("byteOffset"), len(v))
	case []uint16:
		b := s.uint16ToBytes(v)
		uint8 = js.Global().Get("Uint8Array").New(len(b))
		js.CopyBytesToJS(uint8, b)
		return js.Global().Get("Uint16Array").New(uint8.Get("buffer"), uint8.Get("byteOffset"), len(v))
	}
	return js.Null()
}

func (s *WebGPUState) float32ToBytes(f []float32) []byte {
	size := len(f) * 4
	buf := make([]byte, size)
	for i, val := range f {
		bits := math.Float32bits(val)
		buf[i*4] = byte(bits)
		buf[i*4+1] = byte(bits >> 8)
		buf[i*4+2] = byte(bits >> 16)
		buf[i*4+3] = byte(bits >> 24)
	}
	return buf
}

func (s *WebGPUState) uint16ToBytes(u []uint16) []byte {
	size := len(u) * 2
	buf := make([]byte, size)
	for i, val := range u {
		buf[i*2] = byte(val)
		buf[i*2+1] = byte(val >> 8)
	}
	return buf
}

func (s *WebGPUState) Render(angle float32, width, height int) {
	if s.depthTexture.Type() != js.TypeObject || s.depthTexture.Get("width").Int() != width || s.depthTexture.Get("height").Int() != height {
		if s.depthTexture.Type() == js.TypeObject {
			s.depthTexture.Call("destroy")
		}
		s.depthTexture = s.device.Call("createTexture", map[string]any{
			"size":   []any{width, height},
			"format": "depth24plus",
			"usage":  0x10,
		})
	}

	aspect := float32(width) / float32(height)
	proj := Mat4Perspective(math.Pi/4, aspect, 0.1, 100)
	view := Mat4LookAt([3]float32{0, 0, 4}, [3]float32{0, 0, 0}, [3]float32{0, 1, 0})
	model := Mat4Multiply(Mat4RotateY(angle), Mat4RotateX(angle*0.7))
	mvp := Mat4Multiply(proj, Mat4Multiply(view, model))

	s.device.Get("queue").Call("writeBuffer", s.uniformBuffer, 0, s.toTypedArray(mvp[:]))

	viewTex := s.context.Call("getCurrentTexture").Call("createView")
	depthView := s.depthTexture.Call("createView")
	encoder := s.device.Call("createCommandEncoder")
	pass := encoder.Call("beginRenderPass", map[string]any{
		"colorAttachments": []any{
			map[string]any{
				"view":       viewTex,
				"clearValue": map[string]any{"r": 0.08, "g": 0.09, "b": 0.12, "a": 1},
				"loadOp":     "clear",
				"storeOp":    "store",
			},
		},
		"depthStencilAttachment": map[string]any{
			"view":              depthView,
			"depthClearValue":   1,
			"depthLoadOp":       "clear",
			"depthStoreOp":      "store",
			"stencilLoadOp":     "clear",
			"stencilStoreOp":    "store",
			"stencilClearValue": 0,
		},
	})

	pass.Call("setPipeline", s.pipeline)
	pass.Call("setBindGroup", 0, s.uniformBindGroup)
	pass.Call("setVertexBuffer", 0, s.vertexBuffer)
	pass.Call("setIndexBuffer", s.indexBuffer, "uint16")
	pass.Call("drawIndexed", s.numIndices)
	pass.Call("end")

	cmd := encoder.Call("finish")
	s.device.Get("queue").Call("submit", []any{cmd})
}

func (s *WebGPUState) SetColor(face int, r, g, b float32) {
	for i := face * 4; i < face*4+4; i++ {
		base := i*6 + 3
		s.vertexData[base] = r
		s.vertexData[base+1] = g
		s.vertexData[base+2] = b
	}
	s.updateVertexBuffer()
}

func buildCubeData() ([]float32, []uint16) {
	faces := [6][4][3]float32{
		{{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1}},
		{{1, -1, -1}, {-1, -1, -1}, {-1, 1, -1}, {1, 1, -1}},
		{{-1, 1, 1}, {1, 1, 1}, {1, 1, -1}, {-1, 1, -1}},
		{{-1, -1, -1}, {1, -1, -1}, {1, -1, 1}, {-1, -1, 1}},
		{{1, -1, 1}, {1, -1, -1}, {1, 1, -1}, {1, 1, 1}},
		{{-1, -1, -1}, {-1, -1, 1}, {-1, 1, 1}, {-1, 1, -1}},
	}
	colors := [6][3]float32{
		{0.86, 0.21, 0.27},
		{0.20, 0.60, 0.86},
		{0.18, 0.80, 0.44},
		{0.95, 0.77, 0.06},
		{0.61, 0.35, 0.71},
		{1.00, 0.60, 0.20},
	}

	var verts []float32
	var idx []uint16
	for f := 0; f < 6; f++ {
		c := colors[f]
		base := uint16(f * 4)
		for _, p := range faces[f] {
			verts = append(verts, p[0], p[1], p[2], c[0], c[1], c[2])
		}
		idx = append(idx, base, base+1, base+2, base, base+2, base+3)
	}
	return verts, idx
}

func BuildCubeDataForTest() ([]float32, []uint16) {
	return buildCubeData()
}

type ColorPreset struct {
	Name    string
	Face    int
	R, G, B float64
}

var Presets = []ColorPreset{
	{"Red Front", 0, 0.86, 0.21, 0.27},
	{"Blue Back", 1, 0.20, 0.60, 0.86},
	{"Green Top", 2, 0.18, 0.80, 0.44},
	{"Gold Bottom", 3, 0.95, 0.77, 0.06},
	{"Purple Right", 4, 0.61, 0.35, 0.71},
	{"Orange Left", 5, 1.00, 0.60, 0.20},
	{"White Front", 0, 1.0, 1.0, 1.0},
}

type AppState struct {
	Speed        func() float64
	SetSpeed     func(float64)
	Supported    func() bool
	SetSupported func(bool)
	ErrMsg       func() string
	SetErrMsg    func(string)
	Running      func() bool
	SetRunning   func(bool)
	FPS          func() int
	SetFPS       func(int)
	Renderer     *WebGPUState
}

func NewAppState() *AppState {
	speed, setSpeed := reactive.NewSignal(1.0)
	supported, setSupported := reactive.NewSignal(true)
	errMsg, setErrMsg := reactive.NewSignal("")
	running, setRunning := reactive.NewSignal(false)
	fps, setFps := reactive.NewSignal(0)

	return &AppState{
		Speed:        speed,
		SetSpeed:     setSpeed,
		Supported:    supported,
		SetSupported: setSupported,
		ErrMsg:       errMsg,
		SetErrMsg:    setErrMsg,
		Running:      running,
		SetRunning:   setRunning,
		FPS:          fps,
		SetFPS:       setFps,
		Renderer:     &WebGPUState{},
	}
}
