# Reporting Dashboard

A data-heavy dashboard with file upload and table rendering. Upload a CSV or JSON file and see its contents rendered in a performant table. Tested with files containing over 1 million rows.

## Run it

```
make serve
```

Open http://localhost:8086. Drag and drop or select a CSV/JSON file to see results in the table.

## How it works

### gu concepts demonstrated

**File input handling** — Uses the browser File API via Go/WASM to read uploaded files and parse them into structured data.

**Efficient table rendering** — Demonstrates rendering large datasets in tables using gu's reactive system, only updating DOM nodes that change.

**Signal-driven state** — Upload state, parsed data, and error messages are all managed as reactive signals that automatically update the UI.

**Layered architecture** — Follows the recommended `state/` → `components/` → `app/` pattern, keeping business logic separate from UI rendering.