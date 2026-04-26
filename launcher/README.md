# Examples Launcher

This is the main entry point for the `gu-examples` repository. It provides a sidebar to navigate between different examples and an iframe to view them.

## Development

To build and serve the launcher along with all examples:

```bash
make serve
```

This will:
1. Build all examples in their respective directories.
2. Build the launcher.
3. Copy all examples and assets to a central `dist/` directory.
4. Start a local HTTP server at http://localhost:8080.

## Architecture

The launcher follows the standard `gu` architecture:
- `main.go`: Mounts the app.
- `styles.go`: Global CSS.
- `app/`: Main application orchestration.
- `components/`: UI components (Sidebar, Viewer, etc.).
- `registry/`: Metadata about available examples.
- `state/`: Reactive state for the launcher.
