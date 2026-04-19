/**
 * gu WASM debug bootstrap: ?DEBUG_CONSOLE / ?GU_DEBUG_CONSOLE_ENABLED, optional
 * GU_DEBUG_CONSOLE_ENABLED injection from gu dev, toast, BroadcastChannel hub
 * (per-tab channel name; debug console URL includes GU_DEBUG_BC).
 * Load after wasm_exec.js, before constructing Go().
 */
(function () {
  function parseTruthy(s) {
    return ["1", "true", "yes", "on"].includes(String(s || "").trim().toLowerCase());
  }
  const params = new URLSearchParams(window.location.search);
  const hasUrlFlag =
    params.has("GU_DEBUG_CONSOLE_ENABLED") || params.has("DEBUG_CONSOLE");
  var enabled;
  if (hasUrlFlag) {
    enabled = parseTruthy(
      params.get("GU_DEBUG_CONSOLE_ENABLED") || params.get("DEBUG_CONSOLE")
    );
  } else {
    var ev = /** @type {any} */ (window).__guDebugConsoleEnabledFromEnv;
    enabled = ev === true || parseTruthy(String(ev != null ? ev : ""));
  }
  window.__guDebugConsoleEnabled = enabled;

  window.__guDebugPreInit = function (go) {
    if (enabled) {
      go.env.DEBUG_CONSOLE = "true";
    }
  };

  window.__guWireDebugMemory = function (instance) {
    if (!enabled) return;
    const mem = instance.exports.mem;
    window.__guWasmMemory = mem;
  };

  window.guWasmMemoryBytes = function () {
    const m = window.__guWasmMemory;
    if (!m || !m.buffer) return 0;
    return m.buffer.byteLength;
  };

  /** wasm32 addressable linear memory ceiling (4 GiB) when the runtime does not publish a lower max. */
  var WASM32_MAX_LINEAR_BYTES = 65536 * 65536;

  /**
   * WASM linear memory upper bound (bytes):
   * - WebAssembly.Memory.maximum × 64 KiB, or
   * - resizable ArrayBuffer maxByteLength, or
   * - wasm32 ceiling (4 GiB) so usage like 6 MiB scales to a small bar instead of 100% of “session peak”.
   */
  window.guWasmMemoryMaxBytes = function () {
    const m = window.__guWasmMemory;
    if (!m) return 0;
    var buf;
    try {
      buf = m.buffer;
    } catch (e) {
      return 0;
    }
    if (!buf) return 0;
    try {
      var pages = m.maximum;
      if (typeof pages === "number" && pages > 0) {
        var maxBytesFromPages = pages * 65536;
        // Some engines report .maximum tied to current allocation, not the module limit.
        // Only trust it if it is meaningfully above the live buffer size.
        if (maxBytesFromPages > buf.byteLength + 65536) {
          return maxBytesFromPages;
        }
      }
    } catch (e1) {}
    try {
      if (typeof buf.maxByteLength === "number" && buf.maxByteLength > 0) {
        return buf.maxByteLength;
      }
    } catch (e2) {}
    return WASM32_MAX_LINEAR_BYTES;
  };

  const MAX_OPS = 8000;
  const MAX_LOGS = 8000;
  const state = { ops: [], logs: [], lastMem: null, goVersion: "" };

  /** One BroadcastChannel name per app tab so the debug console does not mix snapshots from other tabs or stale sessions. */
  function getOrCreateBroadcastName() {
    var w = window;
    if (!w.__guDebugBroadcastName) {
      var id =
        typeof crypto !== "undefined" &&
        crypto.randomUUID &&
        typeof crypto.randomUUID === "function"
          ? crypto.randomUUID().replace(/-/g, "")
          : String(Date.now()) + "-" + String(Math.random()).slice(2, 12);
      w.__guDebugBroadcastName = "gu-debug-v1-" + id;
    }
    return w.__guDebugBroadcastName;
  }

  function trimOps() {
    if (state.ops.length > MAX_OPS) {
      state.ops.splice(0, state.ops.length - MAX_OPS);
    }
  }

  function trimLogs() {
    if (state.logs.length > MAX_LOGS) {
      state.logs.splice(0, state.logs.length - MAX_LOGS);
    }
  }

  let bc = null;
  function channel() {
    if (!enabled) return null;
    if (!bc) bc = new BroadcastChannel(getOrCreateBroadcastName());
    return bc;
  }

  window.guDebugPublish = function (jsonStr) {
    if (!enabled) return;
    let payload;
    try {
      payload = typeof jsonStr === "string" ? JSON.parse(jsonStr) : jsonStr;
    } catch (e) {
      console.warn("guDebugPublish parse", e);
      return;
    }
    if (payload.mem) {
      const cap =
        typeof window.guWasmMemoryMaxBytes === "function" ? window.guWasmMemoryMaxBytes() : 0;
      if (cap > 0) {
        payload.mem.linearMaxBytes = cap;
      }
      state.lastMem = payload.mem;
    }
    if (payload.goVersion) {
      state.goVersion = payload.goVersion;
    }
    if (payload.ops && payload.ops.length) {
      state.ops.push.apply(state.ops, payload.ops);
      trimOps();
    }
    if (payload.logs && payload.logs.length) {
      state.logs.push.apply(state.logs, payload.logs);
      trimLogs();
    }
    const ch = channel();
    if (ch) {
      ch.postMessage({
        type: "delta",
        ops: payload.ops || [],
        logs: payload.logs || [],
        mem: payload.mem || null,
        goVersion: state.goVersion || "",
      });
    }
  };

  function postSnapshot() {
    const ch = channel();
    if (!ch) return;
    ch.postMessage({
      type: "snapshot",
      ops: state.ops.slice(),
      logs: state.logs.slice(),
      mem: state.lastMem,
      goVersion: state.goVersion || "",
    });
  }

  if (enabled) {
    const ch = channel();
    if (ch) {
      ch.onmessage = function (ev) {
        if (ev.data && ev.data.type === "gu_sync") {
          postSnapshot();
        }
      };
    }
  }

  window.__guDebugMountToast = function () {
    if (!enabled) return;
    const bar = document.createElement("div");
    bar.id = "gu-debug-toast";
    bar.style.cssText = [
      "position:fixed",
      "top:0",
      "left:0",
      "right:0",
      "z-index:2147483000",
      "display:flex",
      "align-items:center",
      "justify-content:center",
      "gap:12px",
      "padding:10px 14px",
      "background:#facc15",
      "color:#1c1917",
      "font:600 13px system-ui,sans-serif",
      "box-shadow:0 2px 8px rgba(0,0,0,.2)",
    ].join(";");
    const msg = document.createElement("span");
    msg.textContent = "Debug mode enabled (GU_DEBUG_CONSOLE_ENABLED / DEBUG_CONSOLE)";
    const btn = document.createElement("button");
    btn.type = "button";
    btn.textContent = "Open debug console";
    btn.style.cssText =
      "cursor:pointer;border:none;border-radius:6px;padding:6px 12px;font:inherit;font-weight:700;background:#1c1917;color:#facc15";
    btn.addEventListener("click", function () {
      var parts = ["GU_DEBUG_BC=" + encodeURIComponent(getOrCreateBroadcastName())];
      var e = window.__guSourceRepoFromEnv;
      if (e && e.baseUrl && e.branch) {
        parts.push("GU_DEBUG_SRC_REPO=" + encodeURIComponent(e.baseUrl));
        parts.push("GU_DEBUG_SRC_BRANCH=" + encodeURIComponent(e.branch));
        if (e.localRoot) {
          parts.push("GU_DEBUG_SRC_ROOT=" + encodeURIComponent(e.localRoot));
        }
        if (window.__guGoVersionFromEnv) {
          parts.push(
            "GU_DEBUG_GO_VERSION=" +
              encodeURIComponent(String(window.__guGoVersionFromEnv))
          );
        }
      }
      var qs = "?" + parts.join("&");
      window.open("debug_console.html" + qs, "guDebugConsole", "width=1100,height=820,menubar=no,toolbar=no");
    });
    bar.appendChild(msg);
    bar.appendChild(btn);
    document.body.appendChild(bar);
    document.body.style.paddingTop = (bar.offsetHeight + 4) + "px";
  };

  if (enabled) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", function () {
        window.__guDebugMountToast();
      });
    } else {
      window.__guDebugMountToast();
    }
  }
})();
