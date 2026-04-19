# Logging & Observability

Demonstrates the framework's logging and debugging features including structured logging, stack traces, and the gu debug console for real-time WASM observability.

## Run it

```
make serve
```

Open http://localhost:8086. Interact with the buttons to trigger different log levels and see structured output in the debug console.

## How it works

### gu concepts demonstrated

**Structured logging** — Showcases `jsutil.LogInfo`, `LogWarn`, and `LogError` with field support for rich, filterable log output.

**Debug console** — Integration with the gu debug console for real-time WASM observability, viewing logs and state changes as they happen.

**Stack traces** — Demonstrates capturing and displaying meaningful Go stack traces in the browser, making it easier to debug WASM applications.