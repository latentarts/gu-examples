# Stocks Dashboard

Real-time simulated financial data visualization with a split-pane layout. Add stock panels, resize them by dragging, and watch prices update live with candlestick charts.

## Run it

```
make serve
```

Open http://localhost:8086. Press Ctrl+Space to open the command palette and add stocks. Drag split handles to resize panels.

## How it works

### gu concepts demonstrated

**Complex interactive state** — Multiple signals managing panel selection, command palette, and placement mode simultaneously.

**Dynamic tree layout** — A recursive split-pane system where panels can be split horizontally or vertically, with reactive resizing.

**Canvas rendering via innerHTML** — Uses `el.OnMount` to get a direct DOM reference and updates chart HTML on a timer, bypassing the reactive system for high-frequency updates.

**Keyboard shortcuts** — Global event handlers for Ctrl+Space (palette), Ctrl+X (close), Escape, and arrow keys.

**Command palette pattern** — A searchable dropdown with keyboard navigation, filtering, and selection.