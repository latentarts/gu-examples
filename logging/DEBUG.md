# WASM debug console (`DEBUG_CONSOLE`)

## Environment variables (`gu dev` only reads these)

`gu dev` uses **only** the following names when injecting into served `.html` (shell or `--env` file). No other environment variables affect this behavior.

| Variable | Role |
|----------|------|
| `GU_DEBUG_CONSOLE_ENABLED` | Truthy → inject debug console enable flag |
| `GU_DEBUG_SRC_REPO` | Git web UI base URL for repo links / raw fetch |
| `GU_DEBUG_SRC_BRANCH` | Branch name (with `GU_DEBUG_SRC_REPO`) |
| `GU_DEBUG_SRC_ROOT` or `GU_DEBUG_SRC_PATH` | Same meaning: absolute checkout root = prefix of paths in stacks |
| `GU_DEBUG_GO_VERSION` | Optional override for stdlib GitHub branch mapping (e.g. `go1.26.0`) |

**Not** environment variables: URL query `GU_DEBUG_BC` (BroadcastChannel name), and `DEBUG_CONSOLE` / `GU_DEBUG_CONSOLE_ENABLED` in the **app URL** for enabling tracing (see below).

## Enable

1. **URL** (any static server): add a query flag, for example:

   `http://localhost:8080/?DEBUG_CONSOLE=true`

   Or the explicit name:

   `http://localhost:8080/?GU_DEBUG_CONSOLE_ENABLED=1`

2. **`gu dev` + shell** (no URL flag): set `GU_DEBUG_CONSOLE_ENABLED=1` in the environment when you start `gu dev`. The dev server injects `window.__guDebugConsoleEnabledFromEnv` into every served `.html`; `debug_boot.js` reads it the same way as the URL flags.

   You can load variables from a file instead of exporting them in the shell:

   `gu dev --env .env.gu`

   Use `--env` more than once for layered files: later files override earlier files for the same key. A variable is left unchanged only when it is already set to a **non-blank** value when `gu dev` starts. If a key exists but is empty (or whitespace only), the value from the file is applied so placeholders in the shell do not block `.env`.

   Example `.env.gu`:

   ```
   GU_DEBUG_CONSOLE_ENABLED=1
   GU_DEBUG_SRC_REPO=https://git.example.com/org/gu
   GU_DEBUG_SRC_BRANCH=main
   GU_DEBUG_SRC_ROOT=/abs/path/to/repo
   ```

3. `debug_boot.js` sets `go.env.DEBUG_CONSOLE` before `go.run` when enabled, and shows a yellow toast with **Open debug console**.

4. Click **Open debug console** to open `debug_console.html` in a new window. It receives live event batches over a per-tab `BroadcastChannel` name (`GU_DEBUG_BC` in the URL). That isolates this session from other browser tabs; opening `debug_console.html` by hand without `GU_DEBUG_BC` falls back to the shared default name and can mix traffic if multiple debug-enabled app tabs are open.

If both URL and injection are present, the **URL** wins (so you can force the console off with `?GU_DEBUG_CONSOLE_ENABLED=0` even when the shell has it enabled).

## Build artifacts

The `gu` CLI **embeds** the WASM debug UI (`debug_boot.js`, `debug_console.js`, `debug_console.html`) from `cmd/gu/templates/debug/` in the binary. **`gu build`** always writes those three files next to `index.html` in the output directory. **`gu dev`** writes the same files into the **project root** on startup (same relative URLs as `index.html`). Treat them as generated from your installed `gu` version; add them to `.gitignore` if you do not want them tracked (this example includes `.gitignore` entries).

Use `gu build --debug` (Go only) to omit `-w -s` so stack traces keep full symbol information.

### Release WASM (`gu_notrace`)

By default, `gu build` compiles WASM with **`-tags=gu_notrace`**, which **omits** the `debugutil` tracing implementation and the log-to-debug-console bridge from `main.wasm` entirely (smaller binary; `DEBUG_CONSOLE` cannot enable them in that build). Use **`gu build --devtools`** when you need a `dist/` WASM that still includes those hooks (for example shipping a staging build with the observability window). **`gu dev`** always builds **without** `gu_notrace` so local development keeps the full feature set.

## Go instrumentation

Import `github.com/latentart/gu/debugutil` and wrap work in `debugutil.WithOp("name", func() error { ... })`. In **devtools** WASM builds, when `DEBUG_CONSOLE` is off at runtime, `WithOp` is a no-op aside from calling `fn`. In **`gu_notrace`** release builds, `WithOp` is always the same no-op regardless of environment.

### Structured fields on log lines (`jsutil.LogFields`)

`jsutil.LogDebug`, `LogInfo`, `LogWarn`, and `LogError` accept an optional **`jsutil.LogFields`** value as the **last** argument. It is not passed to `fmt.Sprintf` — use it after every placeholder argument:

```go
jsutil.LogInfo("user %s logged in", email, jsutil.LogFields{"route": "/home", "session": sid})
```

Field values are stringified for the browser console (second grey line under the badge), for the JSON batch sent to `guDebugPublish`, and for the WASM debug console. There, click a **log** row in the event list: the side panel shows the **message** at the top (same layout idea as exceptions), **Formatted** is a **Variable / Value** table, and **Raw** is plain text.

If the format string still has `%` verbs after you strip `LogFields`, you will get the usual missing-argument behaviour from `fmt.Sprintf` — keep `LogFields` last and keep placeholder counts aligned.

## Source snippets (`gu dev` only)

With `DEBUG_CONSOLE` enabled, `jsutil` may request `GET /_debug/source?file=...&line=...` after an exception. The `gu dev` server serves a few lines of any `.go` file **under the current working directory** only (path traversal is rejected).

## Source snippets (repo fallback)

If the stack frame points outside the `gu dev` working directory (for example, if paths are absolute), the debug console can optionally fetch code from a git web UI instead of `/_debug/source`.

### Option A — URL query (any static server)

- `GU_DEBUG_SRC_REPO`: repo base URL (no trailing slash; `.git` suffix is stripped), for example `https://github.com/latentart/gu`
- `GU_DEBUG_SRC_BRANCH`: branch name (example: `main`)
- `GU_DEBUG_SRC_ROOT` or **`GU_DEBUG_SRC_PATH`** (same meaning): optional **absolute** checkout root that is a **prefix** of paths in stack traces, so `/home/you/.../gu/pkg/foo.go` maps to `pkg/foo.go` in the repo. If this does not match the machine that built the WASM binary, mapping fails and repo links stay empty.

The console picks URL shapes from the hostname: **github.com** uses `blob` / `raw.githubusercontent.com`, hostnames containing **`gitlab`** use GitLab `/-/blob` / `/-/raw`, and other hosts (Gitea, Forgejo, etc.) use `/src/branch/...` and **`/raw/branch/...`** for raw text (required for in-page snippet fetch).

Example:

`debug_console.html?GU_DEBUG_SRC_REPO=https%3A%2F%2Fgithub.com%2Flatentart%2Fgu&GU_DEBUG_SRC_BRANCH=main&GU_DEBUG_SRC_ROOT=%2Fhome%2Fyou%2FProjects%2FOSS%2Fprods%2Fgu`

### Option B — environment variables with `gu dev` (recommended)

The browser cannot read your shell environment. When you run **`gu dev`**, it injects into every served `.html` using **only** the variable names in the table at the top of this file.

Example:

`GU_DEBUG_SRC_REPO=https://github.com/latentart/gu GU_DEBUG_SRC_BRANCH=main GU_DEBUG_SRC_ROOT=$PWD/.. gu dev`

### Go standard library (GitHub)

Frames under the Go toolchain (for example `/usr/lib/go/src/runtime/debug/stack.go`) are fetched from **`https://github.com/golang/go`** using `release-branch.go1.NN` derived from the toolchain version.

The WASM app publishes `goVersion` (`runtime.Version()`, e.g. `go1.26.5`) on each debug batch; the console maps that to `release-branch.go1.26`.

Override if needed:

`debug_console.html?GU_DEBUG_GO_VERSION=go1.26.0`
