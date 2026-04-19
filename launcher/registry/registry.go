package registry

// ExampleData holds metadata for a single example application.
type ExampleData struct {
	ID          string
	Name        string
	Description string
	Thumbnail   string
}

// Examples returns the list of all example applications.
func Examples() []ExampleData {
	return []ExampleData{
		{
			ID:          "counter",
			Name:        "Counter",
			Description: "A classic reactive counter demonstrating gu's core primitives — signals for state, memos for derived values, and conditional rendering with el.Show.",
			Thumbnail:   "/assets/counter.png",
		},
		{
			ID:          "duckdb",
			Name:        "DuckDB Explorer",
			Description: "Interactive SQL query editor running entirely in the browser. Type SQL, execute it against an in-memory DuckDB database pre-loaded with sample data, and see results in a styled table.",
			Thumbnail:   "/assets/duckdb.png",
		},
		{
			ID:          "logging",
			Name:        "Logging & Observability",
			Description: "Demonstrates the framework's logging and debugging features including structured logging, stack traces, and the gu debug console for real-time WASM observability.",
			Thumbnail:   "/assets/logging.png",
		},
		{
			ID:          "nodegraph",
			Name:        "Node Graph",
			Description: "A visual, interactive node-based graph editor with drag-and-drop, zooming, panning, and SVG connection lines.",
			Thumbnail:   "/assets/nodegraph.png",
		},
		{
			ID:          "openai-chat",
			Name:        "OpenAI Chat",
			Description: "A streaming chat interface for OpenAI with Server-Sent Events (SSE) and partial updates rendered reactively.",
			Thumbnail:   "/assets/openai-chat.png",
		},
		{
			ID:          "reporting",
			Name:        "Reporting Dashboard",
			Description: "A data-heavy dashboard with file upload and table rendering. Upload CSV or JSON files and view results in a performant table, tested with over 1 million records.",
			Thumbnail:   "/assets/reporting.png",
		},
		{
			ID:          "shadcn",
			Name:        "shadcn/ui Components",
			Description: "Implementation of modern UI components (Buttons, Cards, Inputs) using gu's functional approach combined with utility-first CSS styling.",
			Thumbnail:   "/assets/shadcn.png",
		},
		{
			ID:          "stocks",
			Name:        "Stocks Dashboard",
			Description: "Real-time simulated financial data visualization with split-pane layout, live candlestick charts, and a command palette for adding stocks.",
			Thumbnail:   "/assets/stocks.png",
		},
		{
			ID:          "tailwind",
			Name:        "Tailwind CSS",
			Description: "Showcases seamless use of Tailwind CSS with gu, demonstrating el.Class and el.DynClass to apply utility styles reactively.",
			Thumbnail:   "/assets/tailwindcss.png",
		},
		{
			ID:          "webgpu",
			Name:        "WebGPU Rotating Cube",
			Description: "Animated 3D cube rendered with WebGPU and controlled from Go. The render loop runs via requestAnimationFrame, driven from a Go callback reading reactive signals.",
			Thumbnail:   "/assets/webgpu.png",
		},
		{
			ID:          "webllm",
			Name:        "WebLLM Chat",
			Description: "Chat interface with a language model running entirely in the browser via WebLLM. No server, no API keys — the model downloads and runs in WebGPU/WASM.",
			Thumbnail:   "/assets/webllm.png",
		},
	}
}

// FindByID returns an example by its ID, or nil if not found.
func FindByID(id string) *ExampleData {
	for _, ex := range Examples() {
		if ex.ID == id {
			return &ex
		}
	}
	return nil
}