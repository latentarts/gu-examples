(function () {
  const statusEl = document.getElementById("status");
  const cpuBars = document.getElementById("cpu-bars");
  const ramBars = document.getElementById("ram-bars");
  const timelineEl = document.getElementById("timeline");
  const logEl = document.getElementById("log");
  const logSplit = document.getElementById("log-split");
  const btnDetailToggle = document.getElementById("btn-detail-toggle");
  const btnDetailHide = document.getElementById("btn-detail-hide");
  const logSearchInput = document.getElementById("log-search-input");
  /** @type {Record<string, {event:string, err:string, stack:string, name:string}>} */
  const opDetailsById = Object.create(null);

  let detailPanelVisible = true;
  /** @type {number|string|null} */
  let selectedOpId = null;
  /** @type {number|null} */
  let selectedLogId = null;
  let nextLogRowId = 1;
  let detailTab = "formatted";
  /** @type {AbortController|null} */
  let detailSnippetAbort = null;
  /** From WASM publish (`runtime.Version`) or overridden via `?GU_DEBUG_GO_VERSION=`. */
  let lastGoVersion = "";
  /** Avoid rebuilding the details panel every render tick (stops formatted stack flicker). */
  let lastDetailSyncKey = "";

  const MAX_OPS = 8000;
  let ops = [];
  /**
   * @type {{id:number,t:number,level:string,msg:string,fields?:Record<string,string>|null}[]}
   */
  let traceLogs = [];
  let lastMem = null;
  let maxRamSeen = 1;
  /** WASM linear memory ceiling from main window (bytes), when exposed by the runtime. */
  let linearMaxCap = 0;
  /** wasm32 linear memory ceiling (matches debug_boot fallback). */
  const WASM32_MAX_LINEAR_BYTES = 65536 * 65536;
  /** @type {{t:number, wasm:number, heap:number}[]} */
  let memSeries = [];
  const bcQuery = new URLSearchParams(window.location.search).get("GU_DEBUG_BC");
  const bcName =
    bcQuery && String(bcQuery).trim() !== "" ? String(bcQuery).trim() : "gu-debug-v1";
  const bc = new BroadcastChannel(bcName);

  let paused = false;
  let pauseWallStart = 0;
  /** @type {{t0:number,t1:number}[]} */
  let pauseGaps = [];
  /** @type {any[]} */
  let pendingWhilePaused = [];
  let tickRender = null;
  let tickSync = null;

  function hashHue(name) {
    let h = 0;
    for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) | 0;
    return Math.abs(h) % 360;
  }

  function segClass(op) {
    if (op.err) return "seg err";
    const d = op.t1 - op.t0;
    if (d > 16) return "seg slow";
    return "seg";
  }

  function segClassForOp(op) {
    if (op.event === "panic") return "seg panic";
    if (op.event === "exception") return "seg err";
    return segClass(op);
  }

  function segStyle(op) {
    if (op.err || op.event) return "";
    const d = op.t1 - op.t0;
    if (d > 16) return "";
    const hue = hashHue(op.name);
    return "background:hsl(" + hue + ",55%,42%)";
  }

  function recordMem(mem) {
    if (!mem) return;
    lastMem = mem;
    const r = mem.wasmBytes || 0;
    const heap = mem.heapAlloc || 0;
    const cap = mem.linearMaxBytes || 0;
    if (cap > 0) {
      linearMaxCap = Math.max(linearMaxCap, cap);
    }
    maxRamSeen = Math.max(maxRamSeen, r, 1);
    memSeries.push({ t: Date.now(), wasm: r, heap: heap });
    if (memSeries.length > 800) memSeries.splice(0, memSeries.length - 800);
  }

  /** Max from payloads (main window); 0 if not yet sent. */
  function transmittedLinearCap() {
    let c = linearMaxCap;
    if (lastMem && typeof lastMem.linearMaxBytes === "number" && lastMem.linearMaxBytes > 0) {
      c = Math.max(c, lastMem.linearMaxBytes);
    }
    return c;
  }

  /**
   * Effective linear-memory scale for purple bars and labels.
   * Never falls back to session peak (that made ~6 MiB look like 100%).
   */
  function wasmLinearCapBytes() {
    const t = transmittedLinearCap();
    if (t > 0) return t;
    return WASM32_MAX_LINEAR_BYTES;
  }

  function ramHeightDenominator() {
    return wasmLinearCapBytes();
  }

  function ramScaleNote() {
    const t = transmittedLinearCap();
    const eff = wasmLinearCapBytes();
    const base =
      t > 0
        ? "Bar height ÷ reported WASM linear max " + formatMemory(eff)
        : "Bar height ÷ wasm32 4 GiB reference (runtime max matched current size, so peak scaling is misleading)";
    return base + ". Nonzero usage under 1% of max is drawn as a 1% bar.";
  }

  function updateRamCapLabel() {
    const el = document.getElementById("ram-max-inline");
    if (!el) return;
    el.textContent = "· max " + formatMemory(wasmLinearCapBytes());
    el.title =
      transmittedLinearCap() > 0
        ? "WASM linear max from the app window (engine or wasm32 fallback)."
        : "Using wasm32 4 GiB reference because the reported Memory.maximum matched the current buffer.";
  }

  /**
   * @param {unknown} raw
   * @returns {Record<string,string>|null}
   */
  function normalizeLogFields(raw) {
    if (raw == null || typeof raw !== "object" || Array.isArray(raw)) {
      return null;
    }
    /** @type {Record<string,string>} */
    const out = Object.create(null);
    const keys = Object.keys(raw);
    for (let i = 0; i < keys.length; i++) {
      const k = keys[i];
      const v = raw[k];
      if (v != null && typeof v === "object") {
        try {
          out[k] = JSON.stringify(v);
        } catch (e) {
          out[k] = String(v);
        }
      } else {
        out[k] = v == null ? "" : String(v);
      }
    }
    return Object.keys(out).length ? out : null;
  }

  function ingestDelta(msg, skipRender) {
    if (msg.goVersion) lastGoVersion = String(msg.goVersion);
    if (msg.mem) recordMem(msg.mem);
    if (msg.ops && msg.ops.length) {
      ops.push.apply(ops, msg.ops);
      if (ops.length > MAX_OPS) ops.splice(0, ops.length - MAX_OPS);
    }
    if (msg.logs && msg.logs.length) {
      for (let i = 0; i < msg.logs.length; i++) {
        const L = msg.logs[i];
        if (L && typeof L.msg === "string") {
          traceLogs.push({
            id: nextLogRowId++,
            t: typeof L.t === "number" ? L.t : Date.now(),
            level: String(L.level || "INFO"),
            msg: L.msg,
            fields: normalizeLogFields(L.fields),
          });
        }
      }
      if (traceLogs.length > MAX_OPS) traceLogs.splice(0, traceLogs.length - MAX_OPS);
    }
    if (!skipRender) render();
  }

  function ingestSnapshot(msg, skipRender) {
    if (msg.goVersion) lastGoVersion = String(msg.goVersion);
    ops = (msg.ops && msg.ops.slice()) || [];
    traceLogs = [];
    if (msg.logs && msg.logs.length) {
      for (let i = 0; i < msg.logs.length; i++) {
        const L = msg.logs[i];
        if (L && typeof L.msg === "string") {
          traceLogs.push({
            id: nextLogRowId++,
            t: typeof L.t === "number" ? L.t : Date.now(),
            level: String(L.level || "INFO"),
            msg: L.msg,
            fields: normalizeLogFields(L.fields),
          });
        }
      }
      if (traceLogs.length > MAX_OPS) traceLogs.splice(0, traceLogs.length - MAX_OPS);
    }
    if (msg.mem) recordMem(msg.mem);
    if (!skipRender) render();
  }

  function refreshStatusLine() {
    let s;
    if (paused) {
      s =
        "paused · " +
        pendingWhilePaused.length +
        " update(s) queued" +
        (pauseGaps.length ? " · " + pauseGaps.length + " pause break(s) in timeline" : "");
    } else {
      s =
        "live · " +
        (ops.length + traceLogs.length) +
        " events" +
        (lastMem
          ? " · linear " +
            formatMemory(lastMem.wasmBytes || 0) +
            " / max " +
            formatMemory(wasmLinearCapBytes()) +
            (lastMem.heapAlloc ? " · heap " + formatMemory(lastMem.heapAlloc) : "")
          : "");
    }
    statusEl.textContent = s;
  }

  bc.onmessage = function (ev) {
    const d = ev.data;
    if (!d) return;
    if (paused) {
      pendingWhilePaused.push(d);
      refreshStatusLine();
      return;
    }
    if (d.type === "delta") ingestDelta(d);
    else if (d.type === "snapshot") ingestSnapshot(d);
  };

  function stopIntervals() {
    if (tickRender != null) {
      clearInterval(tickRender);
      tickRender = null;
    }
    if (tickSync != null) {
      clearInterval(tickSync);
      tickSync = null;
    }
  }

  function startIntervals() {
    stopIntervals();
    tickSync = setInterval(function () {
      if (paused) return;
      bc.postMessage({ type: "gu_sync" });
    }, 4000);
    tickRender = setInterval(function () {
      if (paused) return;
      render();
    }, 900);
  }

  function pauseLive() {
    if (paused) return;
    paused = true;
    pauseWallStart = Date.now();
    stopIntervals();
    render();
  }

  function resumeLive() {
    if (!paused) return;
    const t1 = Date.now();
    if (pauseWallStart > 0 && t1 > pauseWallStart) {
      pauseGaps.push({ t0: pauseWallStart, t1: t1 });
    }
    pauseWallStart = 0;
    paused = false;
    for (let i = 0; i < pendingWhilePaused.length; i++) {
      const d = pendingWhilePaused[i];
      if (d.type === "delta") ingestDelta(d, true);
      else if (d.type === "snapshot") ingestSnapshot(d, true);
    }
    pendingWhilePaused = [];
    startIntervals();
    render();
  }

  function updatePauseButtons() {
    const bp = document.getElementById("btn-pause");
    const br = document.getElementById("btn-resume");
    if (bp) bp.disabled = paused;
    if (br) br.disabled = !paused;
  }

  /** Binary IEC units (typical for memory). */
  function formatMemory(n) {
    const x = Number(n) || 0;
    if (x >= 1024 * 1024 * 1024) return (x / 1024 / 1024 / 1024).toFixed(2) + " GiB";
    if (x >= 1024 * 1024) return (x / 1024 / 1024).toFixed(2) + " MiB";
    if (x >= 1024) return (x / 1024).toFixed(1) + " KiB";
    return x + " B";
  }

  /** Very short label for under-bar ticks. */
  function formatMemoryShort(n) {
    const x = Number(n) || 0;
    if (x >= 1024 * 1024 * 1024) return (x / 1024 / 1024 / 1024).toFixed(1) + "G";
    if (x >= 1024 * 1024) return (x / 1024 / 1024).toFixed(1) + "M";
    if (x >= 1024) return Math.round(x / 1024) + "K";
    return String(x);
  }

  function formatCpuDelta(ms) {
    const m = Number(ms) || 0;
    if (m >= 1000) return (m / 1000).toFixed(2) + " s";
    if (m >= 1) return m.toFixed(0) + " ms";
    return "<1 ms";
  }

  function formatCpuShort(ms) {
    const m = Number(ms) || 0;
    if (m >= 1000) return (m / 1000).toFixed(1) + "s";
    if (m > 0) return Math.round(m) + "ms";
    return "—";
  }

  function formatWallClock(ms) {
    const d = new Date(ms);
    return d.toLocaleTimeString(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }

  /** Wall-clock start for log rows (local). */
  function formatLogStart(ms) {
    const d = new Date(ms);
    try {
      return d.toLocaleString(undefined, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
    } catch (e) {
      return formatWallClock(ms);
    }
  }

  function appendLogHeader(frag) {
    const hdr = document.createElement("div");
    hdr.className = "log-header";
    const labels = ["#", "Event", "Type", "Δt", "Start", "CPU Δ", "Memory", "Message"];
    for (let i = 0; i < labels.length; i++) {
      const c = document.createElement("div");
      c.className = "log-hdr-cell";
      c.textContent = labels[i];
      hdr.appendChild(c);
    }
    frag.appendChild(hdr);
  }

  function buildMergedLogRows() {
    /** @type {{kind:string,t:number,op?:any,log?:{id:number,t:number,level:string,msg:string,fields?:Record<string,string>|null}}} */
    const rows = [];
    for (let i = 0; i < ops.length; i++) {
      rows.push({ kind: "op", t: ops[i].t0 || 0, op: ops[i] });
    }
    for (let i = 0; i < traceLogs.length; i++) {
      const L = traceLogs[i];
      rows.push({ kind: "log", t: L.t || 0, log: L });
    }
    rows.sort(function (a, b) {
      return a.t - b.t;
    });
    return rows;
  }

  function logSearchTokens() {
    const raw = (logSearchInput ? String(logSearchInput.value) : "").trim().toLowerCase();
    if (!raw) return null;
    const parts = raw.split(/\s+/).filter(Boolean);
    return parts.length ? parts : null;
  }

  /**
   * @param {{kind:string,t:number,op?:any,log?:object}} entry
   * @param {string[]|null} tokens
   */
  function mergedRowMatchesSearch(entry, tokens) {
    if (!tokens) return true;
    const parts = [];
    if (entry.kind === "log" && entry.log) {
      const L = entry.log;
      parts.push("log", String(L.level || ""), String(L.msg || ""));
      if (L.fields && typeof L.fields === "object") {
        const fk = Object.keys(L.fields);
        for (let i = 0; i < fk.length; i++) {
          const k = fk[i];
          parts.push(k, String(L.fields[k]));
        }
      }
    } else if (entry.op) {
      const o = entry.op;
      const wall = o.t1 - o.t0;
      const cpuMs = typeof o.cpuMs === "number" ? o.cpuMs : wall;
      parts.push(String(o.name || ""), String(o.id), String(o.err || ""), String(o.stack || ""));
      parts.push(String(wall), String(cpuMs), formatCpuShort(cpuMs), formatCpuDelta(cpuMs));
      if (typeof o.wasmBytes === "number") {
        parts.push(formatMemory(o.wasmBytes), formatMemoryShort(o.wasmBytes));
      }
      if (typeof o.heapAlloc === "number") {
        parts.push(formatMemory(o.heapAlloc), formatMemoryShort(o.heapAlloc));
      }
      if (o.event === "panic") {
        parts.push("panic", "PANIC");
      } else if (o.event === "exception") {
        parts.push("exception", "EXCEPTION");
      } else {
        parts.push("op", "OP");
      }
    }
    const hay = parts.join("\u0001").toLowerCase();
    for (let i = 0; i < tokens.length; i++) {
      if (hay.indexOf(tokens[i]) === -1) return false;
    }
    return true;
  }

  const LOG_RENDER_MAX = 400;

  /**
   * @returns {{tail:object[], filtered:boolean, totalMatches:number}}
   */
  function getLogRowsForRender() {
    const merged = buildMergedLogRows();
    const tokens = logSearchTokens();
    if (!tokens) {
      return {
        tail: merged.slice(-LOG_RENDER_MAX).reverse(),
        filtered: false,
        totalMatches: merged.length,
      };
    }
    let totalMatches = 0;
    for (let i = 0; i < merged.length; i++) {
      if (mergedRowMatchesSearch(merged[i], tokens)) totalMatches++;
    }
    const tail = [];
    for (let j = merged.length - 1; j >= 0 && tail.length < LOG_RENDER_MAX; j--) {
      if (mergedRowMatchesSearch(merged[j], tokens)) tail.push(merged[j]);
    }
    return { tail: tail, filtered: true, totalMatches: totalMatches };
  }

  function logLevelStripeClass(level) {
    const u = String(level || "INFO").toUpperCase();
    if (u === "DEBUG") return "log-row--log-debug";
    if (u === "INFO") return "log-row--log-info";
    if (u === "WARNING" || u === "WARN") return "log-row--log-warning";
    if (u === "ERROR") return "log-row--log-error";
    return "log-row--log-info";
  }

  function memSampleAtSecond(sec) {
    let wasm = 0;
    let heap = 0;
    for (const p of memSeries) {
      if (Math.floor(p.t / 1000) === sec) {
        wasm = Math.max(wasm, p.wasm || 0);
        heap = Math.max(heap, p.heap || 0);
      }
    }
    if (!wasm && lastMem) {
      wasm = lastMem.wasmBytes || 0;
      heap = lastMem.heapAlloc || 0;
    }
    return { wasm: wasm, heap: heap };
  }

  function formatHmForSec(sec) {
    const d = new Date(sec * 1000);
    return (
      String(d.getHours()).padStart(2, "0") +
      ":" +
      String(d.getMinutes()).padStart(2, "0") +
      ":" +
      String(d.getSeconds()).padStart(2, "0")
    );
  }

  function bucketOps() {
    const m = new Map();
    for (const o of ops) {
      const sec = Math.floor(o.t0 / 1000);
      if (!m.has(sec)) {
        m.set(sec, { cpuMs: 0, byName: new Map(), ops: [] });
      }
      const b = m.get(sec);
      const dur = Math.max(0, o.t1 - o.t0);
      b.cpuMs += dur;
      b.byName.set(o.name, (b.byName.get(o.name) || 0) + dur);
      b.ops.push(o);
    }
    return m;
  }

  function lastSeconds(n) {
    const now = Math.floor(Date.now() / 1000);
    const keys = [];
    for (let i = 0; i < n; i++) keys.push(now - i);
    return keys.reverse();
  }

  function renderBars(keys, bucketMap, kind) {
    const frag = document.createDocumentFragment();
    const n = keys.length;
    for (let idx = 0; idx < n; idx++) {
      const sec = keys[idx];
      const b = bucketMap.get(sec) || { cpuMs: 0, byName: new Map(), ops: [] };
      const wrap = document.createElement("div");
      wrap.className = "bar-wrap";
      const bar = document.createElement("div");
      bar.className = "bar " + (kind === "cpu" ? "cpu" : "ram");
      const clockLabel = formatWallClock(sec * 1000);
      const clockHm = formatHmForSec(sec);
      /** @type {{wasm:number,heap:number}|null} */
      let smRam = null;

      if (kind === "cpu") {
        const cpuMs = b.cpuMs;
        const pctOfSecond = Math.min(100, (cpuMs / 1000) * 100);
        bar.style.height = Math.max(2, (pctOfSecond / 100) * 64) + "px";
        bar.title =
          "Wall clock second: " +
          clockLabel +
          " (" +
          clockHm +
          ")\n" +
          "Traced work in bucket: " +
          formatCpuDelta(cpuMs) +
          "\n" +
          "~" +
          pctOfSecond.toFixed(0) +
          "% of one second if all traced events were serial (not OS CPU utilization)";
      } else {
        smRam = memSampleAtSecond(sec);
        const wasm = smRam.wasm;
        const heap = smRam.heap;
        const denom = ramHeightDenominator();
        const rawPct = denom > 0 ? (wasm / denom) * 100 : 0;
        let hpct = Math.min(100, rawPct || 0);
        if (wasm > 0 && hpct > 0 && hpct < 1) {
          hpct = 1;
        }
        bar.style.height = Math.max(2, (hpct / 100) * 64) + "px";
        bar.title =
          "Wall clock second: " +
          clockLabel +
          " (" +
          clockHm +
          ")\n" +
          "WASM linear memory (max sample in bucket): " +
          formatMemory(wasm) +
          "\n" +
          "Go runtime HeapAlloc (max sample): " +
          formatMemory(heap) +
          "\n" +
          ramScaleNote() +
          "\nActual linear usage: " +
          rawPct.toFixed(3) +
          "% of max · bar height uses " +
          hpct.toFixed(1) +
          "%";
      }

      const tick = document.createElement("div");
      tick.className = "tick";
      const tTime = document.createElement("span");
      tTime.className = "tick-time";
      if (idx % 5 === 0 || idx === n - 1) {
        tTime.textContent = formatHmForSec(sec);
      }
      const tVal = document.createElement("span");
      tVal.className = "tick-val";
      if (kind === "cpu") {
        tVal.textContent = formatCpuShort(b.cpuMs);
      } else {
        tVal.textContent = smRam && smRam.wasm > 0 ? formatMemoryShort(smRam.wasm) : "—";
      }
      tick.appendChild(tTime);
      tick.appendChild(tVal);

      wrap.appendChild(bar);
      wrap.appendChild(tick);
      frag.appendChild(wrap);
    }
    return frag;
  }

  function scrollToOp(id) {
    const row = logEl.querySelector('.log-row[data-row-type="op"][data-op-id="' + id + '"]');
    if (!row) return;
    row.scrollIntoView({ block: "nearest", behavior: "smooth" });
    row.classList.remove("flash");
    void row.offsetWidth;
    row.classList.add("flash");
  }

  /** One horizontal timeline from first traced event → now; each segment is [t0,t1] on wall clock. */
  function renderSessionTimeline() {
    const frag = document.createDocumentFragment();
    const head = document.createElement("div");
    head.className = "session-head";
    const title = document.createElement("span");
    title.className = "session-title";
    title.textContent = "Events timeline";
    const meta = document.createElement("span");
    meta.className = "session-meta mono";
    head.appendChild(title);
    head.appendChild(meta);

    const now = Date.now();
    const hasOps = ops.length > 0;
    const hasGaps = pauseGaps.length > 0;
    if (!hasOps && !hasGaps) {
      meta.textContent = "no events yet";
      frag.appendChild(head);
      const empty = document.createElement("div");
      empty.className = "session-strip";
      empty.style.opacity = "0.5";
      frag.appendChild(empty);
      return frag;
    }

    let start = Infinity;
    let end = now;
    if (hasOps) {
      for (let i = 0; i < ops.length; i++) {
        const o = ops[i];
        if (o.t0 < start) start = o.t0;
        if (o.t1 > end) end = o.t1;
      }
    }
    for (let gi = 0; gi < pauseGaps.length; gi++) {
      const g = pauseGaps[gi];
      if (g.t0 < start) start = g.t0;
      if (g.t1 > end) end = g.t1;
    }
    if (start === Infinity) start = now - 1;
    const range = Math.max(1, end - start);

    const metaParts = [
      formatWallClock(start),
      " → ",
      formatWallClock(end),
      " · span ",
      String(range),
      " ms",
    ];
    if (hasOps) metaParts.push(" · ", String(ops.length), " events");
    if (pauseGaps.length) metaParts.push(" · ", String(pauseGaps.length), " viewer pause(s)");
    if (paused) metaParts.push(" · updates paused");
    meta.textContent = metaParts.join("");

    const strip = document.createElement("div");
    strip.className = "session-strip";

    for (let gi = 0; gi < pauseGaps.length; gi++) {
      const g = pauseGaps[gi];
      const dur = Math.max(0, g.t1 - g.t0);
      const leftPct = ((g.t0 - start) / range) * 100;
      const widthPct = Math.max(0.06, (dur / range) * 100);
      const gapEl = document.createElement("div");
      gapEl.className = "tl-pause-gap";
      gapEl.style.left = leftPct + "%";
      gapEl.style.width = widthPct + "%";
      gapEl.title =
        "Debug console paused (no live updates)\n" +
        formatWallClock(g.t0) +
        " – " +
        formatWallClock(g.t1) +
        "\n" +
        dur +
        " ms";
      if (widthPct > 5) gapEl.textContent = "paused";
      strip.appendChild(gapEl);
    }

    const sorted = ops.slice().sort(function (a, b) {
      if (a.t0 !== b.t0) return a.t0 - b.t0;
      return a.id - b.id;
    });

    for (let i = 0; i < sorted.length; i++) {
      const o = sorted[i];
      const dur = Math.max(0, o.t1 - o.t0);
      const leftPct = ((o.t0 - start) / range) * 100;
      const widthPct = Math.max(0.06, (dur / range) * 100);
      const seg = document.createElement("div");
      seg.className = segClassForOp(o);
      const z = 20 + (o.id % 800);
      let st =
        "left:" +
        leftPct +
        "%;width:" +
        widthPct +
        "%;z-index:" +
        z +
        ";";
      if (!o.err && !o.event && dur <= 16) {
        st += segStyle(o);
      }
      seg.setAttribute("style", st);
      const label = o.name.length > 18 ? o.name.slice(0, 16) + "…" : o.name;
      seg.title =
        o.name +
        "\n#" +
        o.id +
        " · " +
        dur +
        " ms\n" +
        formatWallClock(o.t0) +
        " – " +
        formatWallClock(o.t1) +
        (o.err ? "\n" + o.err : "") +
        (o.event ? "\n\nDetails open in the side panel when selected" : "");
      if (widthPct > 4) seg.textContent = label;
      seg.addEventListener("click", function () {
        showDetailForOp(o);
        scrollToOp(o.id);
      });
      strip.appendChild(seg);
    }

    frag.appendChild(head);
    frag.appendChild(strip);
    return frag;
  }

  function renderLog() {
    const frag = document.createDocumentFragment();
    appendLogHeader(frag);
    const { tail, filtered, totalMatches } = getLogRowsForRender();
    const countEl = document.getElementById("log-search-count");
    if (countEl) {
      if (filtered) {
        countEl.hidden = false;
        countEl.textContent =
          totalMatches === 0
            ? "No matches"
            : "Showing " +
              tail.length +
              " of " +
              totalMatches +
              " match" +
              (totalMatches === 1 ? "" : "es");
      } else {
        countEl.hidden = true;
        countEl.textContent = "";
      }
    }
    for (let i = 0; i < tail.length; i++) {
      const entry = tail[i];
      const row = document.createElement("div");
      row.className = "log-row";

      if (entry.kind === "log" && entry.log) {
        const L = entry.log;
        row.dataset.rowType = "log";
        row.dataset.logId = String(L.id);
        row.classList.add("log-row--log", logLevelStripeClass(L.level));
        row.title =
          "Console log line (not a traced span)" +
          (L.fields && Object.keys(L.fields).length
            ? " — click for message and structured fields"
            : " — click for message");
        if (selectedLogId != null && String(L.id) === String(selectedLogId)) {
          row.classList.add("log-row--selected");
        }
        const id = document.createElement("div");
        id.className = "mono";
        id.textContent = "—";
        const name = document.createElement("div");
        name.className = "log-name-cell";
        name.textContent = "log";
        const typ = document.createElement("div");
        typ.className = "log-type";
        typ.textContent = String(L.level || "INFO");
        const dt = document.createElement("div");
        dt.className = "mono";
        dt.textContent = "—";
        const start = document.createElement("div");
        start.className = "mono";
        start.textContent = formatLogStart(L.t);
        const cpu = document.createElement("div");
        cpu.className = "mono";
        cpu.textContent = "—";
        const mem = document.createElement("div");
        mem.className = "mono";
        mem.textContent = "—";
        const msgCell = document.createElement("div");
        msgCell.className = "mono log-msg-cell";
        msgCell.textContent = L.msg;
        row.appendChild(id);
        row.appendChild(name);
        row.appendChild(typ);
        row.appendChild(dt);
        row.appendChild(start);
        row.appendChild(cpu);
        row.appendChild(mem);
        row.appendChild(msgCell);
        frag.appendChild(row);
        continue;
      }

      const o = entry.op;
      if (!o) continue;
      row.dataset.rowType = "op";
      row.dataset.opId = String(o.id);
      if (o.event === "panic" || o.event === "exception") {
        row.classList.add("log-row--event");
        row.classList.add(o.event === "panic" ? "log-row--panic" : "log-row--exception");
        row.title = "Exception / panic — click for full message and stack in the side panel";
      } else {
        row.classList.add("log-row--plain-op");
        row.title = "Click to show details in the side panel";
      }
      if (selectedOpId != null && String(o.id) === String(selectedOpId)) {
        row.classList.add("log-row--selected");
      }
      const id = document.createElement("div");
      id.className = "mono";
      id.textContent = "#" + o.id;
      const name = document.createElement("div");
      name.textContent = o.name || "";
      const typ = document.createElement("div");
      typ.className = "log-type";
      if (o.event === "panic") {
        typ.classList.add("log-type--panic");
        typ.textContent = "PANIC";
      } else if (o.event === "exception") {
        typ.classList.add("log-type--exception");
        typ.textContent = "EXCEPTION";
      } else {
        typ.classList.add("log-type--op");
        typ.textContent = "OP";
      }
      const dt = document.createElement("div");
      dt.className = "mono";
      dt.textContent = o.t1 - o.t0 + " ms";
      const start = document.createElement("div");
      start.className = "mono";
      start.textContent = formatLogStart(o.t0);
      const cpu = document.createElement("div");
      cpu.className = "mono";
      const cpuMs = typeof o.cpuMs === "number" ? o.cpuMs : o.t1 - o.t0;
      cpu.textContent = formatCpuShort(cpuMs);
      cpu.title = "Traced wall time for this event (same sense as CPU Δ chart): " + formatCpuDelta(cpuMs);
      const mem = document.createElement("div");
      mem.className = "mono";
      const hasWasm = typeof o.wasmBytes === "number";
      const hasHeap = typeof o.heapAlloc === "number";
      if (hasWasm || hasHeap) {
        const wb = hasWasm ? o.wasmBytes : 0;
        const hb = hasHeap ? o.heapAlloc : 0;
        mem.textContent = formatMemoryShort(wb) + " · " + formatMemoryShort(hb);
        mem.title =
          "At span end — WASM linear: " +
          (hasWasm ? formatMemory(wb) : "—") +
          "\nGo heap (runtime.Alloc): " +
          (hasHeap ? formatMemory(hb) : "—");
      } else {
        mem.textContent = "—";
        mem.title = "Not recorded (trace from older gu build)";
      }
      const msgCell = document.createElement("div");
      msgCell.className = "mono log-msg-cell";
      if (o.err) msgCell.classList.add("log-msg-cell--err");
      msgCell.textContent = o.err || "";
      row.appendChild(id);
      row.appendChild(name);
      row.appendChild(typ);
      row.appendChild(dt);
      row.appendChild(start);
      row.appendChild(cpu);
      row.appendChild(mem);
      row.appendChild(msgCell);
      frag.appendChild(row);
    }
    return frag;
  }

  function refreshEventDetailsMap() {
    Object.keys(opDetailsById).forEach(function (k) {
      delete opDetailsById[k];
    });
    for (let i = 0; i < ops.length; i++) {
      const o = ops[i];
      if (o.event === "panic" || o.event === "exception") {
        opDetailsById[String(o.id)] = {
          event: o.event,
          err: o.err || "",
          stack: o.stack || "",
          name: o.name || o.event,
        };
      }
    }
  }

  const goKeywords = new Set([
    "break",
    "case",
    "chan",
    "const",
    "continue",
    "default",
    "defer",
    "else",
    "fallthrough",
    "for",
    "func",
    "go",
    "goto",
    "if",
    "import",
    "interface",
    "map",
    "package",
    "range",
    "return",
    "select",
    "struct",
    "switch",
    "type",
    "var",
  ]);

  function escapeHtml(s) {
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function vscodeFileURL(absPath, line) {
    const col = 1;
    const p = String(absPath).replace(/\\/g, "/");
    if (/^[a-zA-Z]:\//.test(p)) {
      return "vscode://file/" + p + ":" + line + ":" + col;
    }
    if (p.startsWith("/")) {
      return "vscode://file" + p + ":" + line + ":" + col;
    }
    return "vscode://file/" + p + ":" + line + ":" + col;
  }

  function httpDebugSourceURL(filePath, line) {
    const o = window.location.origin;
    const base = !o || o === "null" ? "" : o;
    return base + "/_debug/source?file=" + encodeURIComponent(filePath) + "&line=" + String(line);
  }

  function sourceRepoCfg() {
    const c = /** @type {any} */ (window.__guSourceRepo);
    if (!c || !c.baseUrl || !c.branch) return null;
    return {
      baseUrl: String(c.baseUrl).replace(/\/+$/, "").replace(/\.git$/i, ""),
      branch: String(c.branch),
      localRoot: c.localRoot ? String(c.localRoot) : "",
    };
  }

  function normalizeFsPath(p) {
    return String(p || "")
      .replace(/\\/g, "/")
      .replace(/\/+$/, "");
  }

  /** @param {string} baseUrl */
  function sourceHostingKind(baseUrl) {
    let h = "";
    try {
      h = new URL(baseUrl).hostname.toLowerCase();
    } catch (e) {
      return "gitea";
    }
    if (h === "github.com" || h.endsWith(".github.com")) return "github";
    if (h.indexOf("gitlab") !== -1) return "gitlab";
    return "gitea";
  }

  function encodeBranchSegment(branch) {
    return encodeURIComponent(String(branch || "").trim());
  }

  function encodeRepoRelPathSegments(relPath) {
    return String(relPath || "")
      .split("/")
      .filter(function (s) {
        return s.length > 0;
      })
      .map(encodeURIComponent)
      .join("/");
  }

  function mapAbsPathToRepoPath(absPath) {
    const cfg = sourceRepoCfg();
    if (!cfg) return "";
    const p = normalizeFsPath(absPath);
    const root = normalizeFsPath(cfg.localRoot);
    if (root) {
      if (p === root) return "";
      const prefix = root + "/";
      if (p.startsWith(prefix)) return p.slice(prefix.length);
    }
    // Heuristic fallback for paths that already contain a repo-ish segment.
    const m = p.match(/\/prods\/gu\/(.+)$/);
    if (m) return m[1];
    return "";
  }

  function repoViewURL(absPath) {
    const cfg = sourceRepoCfg();
    if (!cfg) return "";
    const rp = mapAbsPathToRepoPath(absPath);
    if (!rp) return "";
    const base = cfg.baseUrl.replace(/\/+$/, "");
    const br = encodeBranchSegment(cfg.branch);
    const rel = encodeRepoRelPathSegments(rp);
    const kind = sourceHostingKind(base);
    if (kind === "github") {
      return base + "/blob/" + br + "/" + rel;
    }
    if (kind === "gitlab") {
      return base + "/-/blob/" + br + "/" + rel;
    }
    return base + "/src/branch/" + br + "/" + rel;
  }

  function repoRawURL(absPath) {
    const cfg = sourceRepoCfg();
    if (!cfg) return "";
    const rp = mapAbsPathToRepoPath(absPath);
    if (!rp) return "";
    const base = cfg.baseUrl.replace(/\/+$/, "");
    const br = encodeBranchSegment(cfg.branch);
    const rel = encodeRepoRelPathSegments(rp);
    const kind = sourceHostingKind(base);
    if (kind === "github") {
      try {
        const u = new URL(base);
        const parts = u.pathname.replace(/^\/+|\/+$/g, "").split("/").filter(Boolean);
        if (parts.length >= 2) {
          const owner = parts[0];
          const repo = parts[1];
          return (
            "https://raw.githubusercontent.com/" +
            owner +
            "/" +
            repo +
            "/" +
            br +
            "/" +
            rel
          );
        }
      } catch (e) {}
      return "";
    }
    if (kind === "gitlab") {
      return base + "/-/raw/" + br + "/" + rel;
    }
    return base + "/raw/branch/" + br + "/" + rel;
  }

  /**
   * Toolchain string for stdlib GitHub mapping. Prefer WASM runtime.Version() when it
   * parses to a release branch, so a mis-set GU_DEBUG_GO_VERSION (e.g. "1.26") does not
   * override a good value from the app.
   */
  function getEffectiveGoVersion() {
    const fromWasm = String(lastGoVersion || "").trim();
    const injected = String(/** @type {any} */ (window).__guGoVersion || "").trim();
    if (goReleaseBranchFromVersion(fromWasm)) return fromWasm;
    if (goReleaseBranchFromVersion(injected)) return injected;
    return fromWasm || injected;
  }

  /**
   * Maps runtime.Version() (e.g. go1.26.0) or loose tags (1.26, v1.26.5) to golang/go
   * release branch names (e.g. release-branch.go1.26).
   */
  function goReleaseBranchFromVersion(ver) {
    const s = String(ver || "").trim();
    if (!s) return "";
    let m = s.match(/go(\d+)\.(\d+)/i);
    if (m) return "release-branch.go" + m[1] + "." + m[2];
    m = s.match(/^v?(\d+)\.(\d+)/);
    if (m) return "release-branch.go" + m[1] + "." + m[2];
    return "";
  }

  /**
   * Path under GOROOT/src for a stack frame (e.g. runtime/debug/stack.go).
   */
  function stdlibRelPathFromAbs(absPath) {
    const p = String(absPath).replace(/\\/g, "/");
    const prefs = [
      "/usr/lib/go/src/",
      "/usr/local/go/src/",
      "/snap/go/current/src/",
      "/opt/homebrew/opt/go/libexec/src/",
    ];
    for (let i = 0; i < prefs.length; i++) {
      const pre = prefs[i];
      if (p.startsWith(pre)) return p.slice(pre.length);
    }
    const libexec = "/libexec/src/";
    const ix = p.indexOf(libexec);
    if (ix !== -1) {
      const rest = p.slice(ix + libexec.length);
      if (/\.go$/.test(rest)) return rest;
    }
    const goSrc = "/go/src/";
    const j = p.indexOf(goSrc);
    if (j !== -1) {
      const rest = p.slice(j + goSrc.length);
      if (/\.go$/.test(rest) && rest.indexOf("/vendor/") === -1) return rest;
    }
    return "";
  }

  function stdlibRawURL(absPath) {
    const rel = stdlibRelPathFromAbs(absPath);
    if (!rel) return "";
    const br = goReleaseBranchFromVersion(getEffectiveGoVersion());
    if (!br) return "";
    return "https://raw.githubusercontent.com/golang/go/" + br + "/src/" + rel;
  }

  function stdlibViewURL(absPath, line) {
    const rel = stdlibRelPathFromAbs(absPath);
    if (!rel) return "";
    const br = goReleaseBranchFromVersion(getEffectiveGoVersion());
    if (!br) return "";
    const ln = line ? "#L" + String(line) : "";
    return "https://github.com/golang/go/blob/" + br + "/src/" + rel + ln;
  }

  function formatTruncPath(fullPath, maxLen) {
    const fp = String(fullPath);
    if (fp.length <= maxLen) return fp;
    return "…" + fp.slice(-(maxLen - 1));
  }

  function parseGoStack(text) {
    if (!text || typeof text !== "string") return [];
    const rawLines = text.split(/\r?\n/);
    const frames = [];
    for (let i = 0; i < rawLines.length; i++) {
      const line = rawLines[i];
      const m = line.match(/^(\t| +)(.+?\.go):(\d+)(\s+.*)?$/);
      if (!m) continue;
      let funcLine = "";
      for (let j = i - 1; j >= 0; j--) {
        const prevTrim = rawLines[j].trim();
        if (!prevTrim) continue;
        if (/^goroutine\s+\d+/.test(prevTrim)) break;
        if (/^---\s/.test(prevTrim)) break;
        funcLine = prevTrim;
        break;
      }
      frames.push({
        file: m[2].trim(),
        line: parseInt(m[3], 10),
        suffix: (m[4] || "").trim(),
        function: funcLine,
      });
    }
    return frames;
  }

  function highlightGoLineSimple(code) {
    let s = escapeHtml(code);
    s = s.replace(
      /\b(break|case|chan|const|continue|default|defer|else|fallthrough|for|func|go|goto|if|import|interface|map|package|range|return|select|struct|switch|type|var)\b/g,
      '<span class="stk-go-kw">$1</span>'
    );
    s = s.replace(/\b(\d+)\b/g, '<span class="stk-go-num">$1</span>');
    return s;
  }

  function extractGoIdentifiers(code) {
    const out = [];
    const re = /\b[a-zA-Z_][a-zA-Z0-9_]*\b/g;
    let m;
    while ((m = re.exec(code))) {
      const w = m[0];
      if (goKeywords.has(w)) continue;
      out.push(w);
    }
    return [...new Set(out)].slice(0, 16);
  }

  function renderFormattedFrames(frames, errText, targetEl) {
    targetEl.replaceChildren();
    const note = document.createElement("p");
    note.className = "stk-note";
    note.textContent =
      "Go runtime stacks do not include local variable values. Names below are inferred from the fetched source line; values are not available from the stack alone.";
    targetEl.appendChild(note);
    if (errText) {
      const errEl = document.createElement("div");
      errEl.className = "stk-top-err";
      errEl.textContent = errText;
      targetEl.appendChild(errEl);
    }
    for (let idx = 0; idx < frames.length; idx++) {
      const fr = frames[idx];
      const wrap = document.createElement("div");
      wrap.className = "stk-frame";

      const loc = document.createElement("div");
      loc.className = "stk-loc-line";
      const kw1 = document.createElement("span");
      kw1.className = "stk-meta-kw";
      kw1.textContent = "File ";
      loc.appendChild(kw1);
      const a = document.createElement("a");
      a.className = "stk-file-link";
      a.href = vscodeFileURL(fr.file, fr.line);
      a.target = "_blank";
      a.rel = "noopener noreferrer";
      a.title = fr.file + " — open in editor (VS Code / Cursor)";
      a.textContent = '"' + formatTruncPath(fr.file, 72) + '"';
      loc.appendChild(a);
      const kw2 = document.createElement("span");
      kw2.className = "stk-meta-kw";
      kw2.textContent = ", line ";
      loc.appendChild(kw2);
      const ln = document.createElement("span");
      ln.className = "stk-line-num";
      ln.textContent = String(fr.line);
      loc.appendChild(ln);
      const kw3 = document.createElement("span");
      kw3.className = "stk-meta-kw";
      kw3.textContent = ", in ";
      loc.appendChild(kw3);
      const fn = document.createElement("span");
      fn.className = "stk-func-name";
      fn.textContent = fr.function || "(unknown)";
      loc.appendChild(fn);
      wrap.appendChild(loc);

      const httpA = document.createElement("a");
      httpA.className = "stk-snippet-link";
      httpA.href = httpDebugSourceURL(fr.file, fr.line);
      httpA.target = "_blank";
      httpA.rel = "noopener noreferrer";
      httpA.textContent = "View snippet (local gu dev)";
      wrap.appendChild(httpA);

      const repoU = repoViewURL(fr.file);
      if (repoU) {
        const repoA = document.createElement("a");
        repoA.className = "stk-snippet-link";
        repoA.href = repoU;
        repoA.target = "_blank";
        repoA.rel = "noopener noreferrer";
        repoA.textContent = "Open file (repo)";
        wrap.appendChild(repoA);
      }

      const stdView = stdlibViewURL(fr.file, fr.line);
      if (stdView) {
        const stA = document.createElement("a");
        stA.className = "stk-snippet-link";
        stA.href = stdView;
        stA.target = "_blank";
        stA.rel = "noopener noreferrer";
        stA.textContent = "Open stdlib (GitHub)";
        wrap.appendChild(stA);
      }

      const codeHold = document.createElement("div");
      codeHold.className = "stk-code-hold";
      const codeInner = document.createElement("code");
      codeInner.className = "stk-code-placeholder";
      const ph = document.createElement("span");
      ph.className = "stk-muted";
      ph.textContent = "Loading source line…";
      codeInner.appendChild(ph);
      codeHold.appendChild(codeInner);
      wrap.appendChild(codeHold);

      const treeHold = document.createElement("div");
      treeHold.className = "stk-tree-hold";
      wrap.appendChild(treeHold);
      targetEl.appendChild(wrap);
    }
    enrichFramesWithSnippets(frames, targetEl);
  }

  function snippetLinesAround(fullText, line) {
    const lines = String(fullText || "").split(/\n/);
    const ln = Math.max(1, Number(line) || 1);
    const start = Math.max(0, ln - 4);
    const end = Math.min(lines.length, ln + 3);
    const out = [];
    for (let i = start; i < end; i++) {
      const prefix = i + 1 === ln ? "> " : "  ";
      out.push(prefix + String(i + 1) + "|" + lines[i]);
    }
    return out.join("\n") + "\n";
  }

  function fetchBestSnippet(fr, signal) {
    // Prefer local /_debug/source, then Go stdlib on GitHub (only for GOROOT paths), then app repo.
    return fetch(httpDebugSourceURL(fr.file, fr.line), { signal: signal }).then(function (r) {
      if (r.ok) return r.text();

      const stdRel = stdlibRelPathFromAbs(fr.file);
      if (stdRel) {
        const stdUrl = stdlibRawURL(fr.file);
        if (stdUrl) {
          return fetch(stdUrl, { signal: signal }).then(function (r2) {
            if (!r2.ok) throw new Error("stdlib_http:" + String(r2.status));
            return r2.text().then(function (t2) {
              return snippetLinesAround(t2, fr.line);
            });
          });
        }
        throw new Error("stdlib_no_branch");
      }

      const rr = repoRawURL(fr.file);
      if (!rr) throw new Error("no_repo_raw:" + String(r.status));
      return fetch(rr, { signal: signal }).then(function (r2) {
        if (!r2.ok) throw new Error("repo_http:" + String(r2.status));
        return r2.text().then(function (t2) {
          return snippetLinesAround(t2, fr.line);
        });
      });
    });
  }

  function enrichFramesWithSnippets(frames, targetEl) {
    if (detailSnippetAbort) detailSnippetAbort.abort();
    detailSnippetAbort = new AbortController();
    const signal = detailSnippetAbort.signal;
    const wraps = targetEl.querySelectorAll(".stk-frame");
    for (let idx = 0; idx < frames.length; idx++) {
      const fr = frames[idx];
      const wrap = wraps[idx];
      if (!wrap) continue;
      const codeHold = wrap.querySelector(".stk-code-hold");
      const treeHold = wrap.querySelector(".stk-tree-hold");
      if (!codeHold || !treeHold) continue;
      fetchBestSnippet(fr, signal)
        .then(function (t) {
          if (signal.aborted) return;
          let hitLine = "";
          const lines = t.split(/\n/);
          for (let li = 0; li < lines.length; li++) {
            const mm = lines[li].match(/^>\s*(\d+)\|\s*(.*)$/);
            if (mm) {
              hitLine = mm[2];
              break;
            }
          }
          if (!hitLine) {
            for (let li = 0; li < lines.length; li++) {
              const mm2 = lines[li].match(/^\s*\d+\|\s*(.*)$/);
              if (mm2) {
                hitLine = mm2[1];
                break;
              }
            }
          }
          if (!hitLine) {
            codeHold.innerHTML =
              '<code class="stk-code"><span class="stk-muted">(Snippet not available — open Raw or check gu dev cwd)</span></code>';
            treeHold.replaceChildren();
            return;
          }
          codeHold.innerHTML = '<code class="stk-code">' + highlightGoLineSimple(hitLine) + "</code>";
          const ids = extractGoIdentifiers(hitLine);
          treeHold.replaceChildren();
          ids.forEach(function (id, j) {
            const row = document.createElement("div");
            row.className = "stk-tree-row";
            const art = document.createElement("span");
            art.className = "stk-tree-art";
            art.textContent = j === ids.length - 1 ? "└ " : "├ ";
            const v = document.createElement("span");
            v.className = "stk-tree-var";
            v.textContent = id;
            const u = document.createElement("span");
            u.className = "stk-tree-unav";
            u.textContent = ": (value not in stack)";
            row.appendChild(art);
            row.appendChild(v);
            row.appendChild(u);
            treeHold.appendChild(row);
          });
        })
        .catch(function (err) {
          if (signal.aborted) return;
          const cfg = sourceRepoCfg();
          const stdRel = stdlibRelPathFromAbs(fr.file);
          const br = goReleaseBranchFromVersion(getEffectiveGoVersion());
          const em = err && err.message ? String(err.message) : "";
          let hint;
          if (em === "stdlib_no_branch" || (stdRel && !br)) {
            hint =
              "Snippet fetch failed (stdlib). Need a Go version that maps to golang/go (e.g. go1.26.5 from WASM, or GU_DEBUG_GO_VERSION=go1.26.x). Bare 1.26 is accepted; override only if it parses.";
          } else if (stdRel && em.indexOf("stdlib_http:") === 0) {
            hint =
              "Snippet fetch failed (stdlib raw on GitHub). Check network or use Open stdlib (GitHub); status " +
              em.slice("stdlib_http:".length) +
              ".";
          } else if (cfg && (em.indexOf("repo_http:") === 0 || em.indexOf("no_repo_raw:") === 0)) {
            hint =
              "Snippet fetch failed (app repo raw). Check GU_DEBUG_SRC_REPO / BRANCH / ROOT or PATH, raw URL, and CORS; status " +
              em.replace(/^repo_http:|^no_repo_raw:/, "") +
              ".";
          } else if (cfg) {
            hint =
              "Snippet fetch failed. Local /_debug/source missed; stdlib/repo fallbacks did not return a snippet.";
          } else {
            hint =
              "Snippet fetch failed (local only). Set GU_DEBUG_SRC_REPO, GU_DEBUG_SRC_BRANCH, and GU_DEBUG_SRC_ROOT or GU_DEBUG_SRC_PATH for repo raw fetch, or use Raw / editor link.";
          }
          codeHold.innerHTML =
            '<code class="stk-code"><span class="stk-muted">' + hint + "</span></code>";
          treeHold.replaceChildren();
        });
    }
  }

  function setDetailTab(which) {
    detailTab = which === "raw" ? "raw" : "formatted";
    const tFmt = document.getElementById("detail-tab-fmt");
    const tRaw = document.getElementById("detail-tab-raw");
    const pFmt = document.getElementById("detail-pane-formatted");
    const pRaw = document.getElementById("detail-pane-raw");
    const isFmt = detailTab === "formatted";
    if (tFmt) {
      tFmt.classList.toggle("detail-tab--active", isFmt);
      tFmt.setAttribute("aria-selected", isFmt ? "true" : "false");
    }
    if (tRaw) {
      tRaw.classList.toggle("detail-tab--active", !isFmt);
      tRaw.setAttribute("aria-selected", isFmt ? "false" : "true");
    }
    if (pFmt) pFmt.hidden = !isFmt;
    if (pRaw) pRaw.hidden = isFmt;
  }

  function showDetailEmptyState(msg) {
    const empty = document.getElementById("detail-empty-msg");
    const filled = document.getElementById("detail-filled");
    if (empty) {
      empty.hidden = false;
      empty.textContent =
        msg != null
          ? msg
          : "Select an event in the log or timeline to see details here.";
    }
    if (filled) filled.hidden = true;
  }

  function populateDetailContent(spec) {
    const empty = document.getElementById("detail-empty-msg");
    const filled = document.getElementById("detail-filled");
    const errLine = document.getElementById("detail-err-line");
    const stackSection = document.getElementById("detail-stack-section");
    const plainBody = document.getElementById("detail-plain-body");
    const rawBody = document.getElementById("detail-raw-body");
    const fmtBody = document.getElementById("detail-pane-formatted");

    if (spec.mode === "empty") {
      if (detailSnippetAbort) detailSnippetAbort.abort();
      lastDetailSyncKey = "";
      showDetailEmptyState(spec.message);
      return;
    }
    if (empty) empty.hidden = true;
    if (filled) filled.hidden = false;

    if (spec.mode === "plain") {
      if (detailSnippetAbort) detailSnippetAbort.abort();
      if (errLine) {
        errLine.hidden = true;
        errLine.classList.remove("detail-msg-line");
      }
      if (stackSection) stackSection.hidden = true;
      if (plainBody) {
        plainBody.hidden = false;
        plainBody.textContent = spec.plainText || "";
      }
      return;
    }

    if (spec.mode === "log") {
      if (detailSnippetAbort) detailSnippetAbort.abort();
      if (plainBody) plainBody.hidden = true;
      if (stackSection) stackSection.hidden = false;
      if (errLine) {
        errLine.hidden = false;
        errLine.classList.add("detail-msg-line");
        errLine.textContent = spec.msg || "";
      }
      const rawLines = [spec.msg || ""];
      if (spec.fields && typeof spec.fields === "object") {
        const fk = Object.keys(spec.fields).sort();
        if (fk.length) {
          rawLines.push("");
          for (let i = 0; i < fk.length; i++) {
            const k = fk[i];
            rawLines.push(k + ": " + String(spec.fields[k]));
          }
        }
      }
      if (rawBody) rawBody.textContent = rawLines.join("\n");
      if (fmtBody) {
        fmtBody.replaceChildren();
        const fk =
          spec.fields && typeof spec.fields === "object" ? Object.keys(spec.fields).sort() : [];
        if (fk.length === 0) {
          const p = document.createElement("p");
          p.className = "stk-muted";
          p.textContent = "(No structured fields on this log line.)";
          fmtBody.appendChild(p);
        } else {
          const tbl = document.createElement("table");
          tbl.className = "detail-log-fields";
          const thead = document.createElement("thead");
          const hr = document.createElement("tr");
          const th1 = document.createElement("th");
          th1.textContent = "Variable";
          const th2 = document.createElement("th");
          th2.textContent = "Value";
          hr.appendChild(th1);
          hr.appendChild(th2);
          thead.appendChild(hr);
          tbl.appendChild(thead);
          const tbody = document.createElement("tbody");
          for (let i = 0; i < fk.length; i++) {
            const k = fk[i];
            const tr = document.createElement("tr");
            const td1 = document.createElement("td");
            td1.textContent = k;
            const td2 = document.createElement("td");
            td2.textContent = String(spec.fields[k]);
            tr.appendChild(td1);
            tr.appendChild(td2);
            tbody.appendChild(tr);
          }
          tbl.appendChild(tbody);
          fmtBody.appendChild(tbl);
        }
      }
      setDetailTab(detailTab);
      return;
    }

    if (spec.mode === "stack") {
      if (detailSnippetAbort) detailSnippetAbort.abort();
      if (plainBody) plainBody.hidden = true;
      if (stackSection) stackSection.hidden = false;
      if (errLine) {
        errLine.classList.remove("detail-msg-line");
        const hasErr = !!(spec.errText && String(spec.errText).trim());
        errLine.hidden = !hasErr;
        errLine.textContent = hasErr ? spec.errText : "";
      }
      if (rawBody) rawBody.textContent = spec.rawCombined || "";
      if (fmtBody) {
        const frames = parseGoStack(spec.stackText || "");
        if (frames.length === 0) {
          fmtBody.replaceChildren();
          const p = document.createElement("pre");
          p.className = "stk-fallback-pre";
          p.textContent =
            spec.stackText ||
            "(No stack text or could not parse .go locations — see Raw tab.)";
          fmtBody.appendChild(p);
        } else {
          renderFormattedFrames(frames, "", fmtBody);
        }
      }
      setDetailTab(detailTab);
    }
  }

  function wireDetailTabs() {
    const tabs = document.querySelector(".detail-tabs");
    if (!tabs) return;
    tabs.addEventListener("click", function (ev) {
      const btn = ev.target.closest("[data-tab]");
      if (!btn) return;
      setDetailTab(btn.getAttribute("data-tab") || "formatted");
    });
  }

  function findOpById(idStr) {
    for (let i = ops.length - 1; i >= 0; i--) {
      if (String(ops[i].id) === String(idStr)) return ops[i];
    }
    return null;
  }

  function findLogById(idStr) {
    for (let i = traceLogs.length - 1; i >= 0; i--) {
      if (String(traceLogs[i].id) === String(idStr)) return traceLogs[i];
    }
    return null;
  }

  function buildDetailSyncKeyForLog(L) {
    if (!L) return "";
    const fk =
      L.fields && typeof L.fields === "object" ? JSON.stringify(L.fields) : "";
    return "log:" + String(L.id) + ":" + String((L.msg || "").length) + ":" + fk;
  }

  function fillDetailSideFromLog(L) {
    const titleEl = document.getElementById("detail-side-title");
    if (!titleEl) return;
    titleEl.textContent = String(L.level || "INFO") + " · log";
    populateDetailContent({
      mode: "log",
      msg: L.msg || "",
      fields: L.fields || null,
    });
    lastDetailSyncKey = buildDetailSyncKeyForLog(L);
  }

  function showDetailForLog(L) {
    if (!L) return;
    detailPanelVisible = true;
    applyDetailPanelState();
    selectedOpId = null;
    selectedLogId = L.id;
    setDetailTab("formatted");
    fillDetailSideFromLog(L);
    updateLogRowSelectionClasses();
  }

  function buildPlainEventDetailText(o) {
    const lines = [];
    lines.push("Event #" + o.id + "  " + (o.name || "(unnamed)"));
    lines.push("");
    lines.push("Δt (traced wall): " + (o.t1 - o.t0) + " ms");
    lines.push("Start (local): " + formatLogStart(o.t0));
    lines.push("End (local):   " + formatLogStart(o.t1));
    const cpuMs = typeof o.cpuMs === "number" ? o.cpuMs : o.t1 - o.t0;
    lines.push("CPU (trace): " + formatCpuDelta(cpuMs));
    const hasWasm = typeof o.wasmBytes === "number";
    const hasHeap = typeof o.heapAlloc === "number";
    if (hasWasm || hasHeap) {
      lines.push(
        "Memory at end — WASM linear: " +
          (hasWasm ? formatMemory(o.wasmBytes) : "—") +
          " · Go heap: " +
          (hasHeap ? formatMemory(o.heapAlloc) : "—")
      );
    }
    if (o.err) {
      lines.push("");
      lines.push("--- error ---");
      lines.push(o.err);
    }
    if (o.stack) {
      lines.push("");
      lines.push("--- stack ---");
      lines.push(o.stack);
    }
    return lines.join("\n");
  }

  function buildDetailSyncKey(o) {
    if (!o) return "";
    const cached = opDetailsById[String(o.id)];
    if (cached) {
      const stack = (cached.stack || "").trim();
      if (stack) {
        return (
          "stack:" +
          String(o.id) +
          ":" +
          String(stack.length) +
          ":" +
          String((cached.err || "").length) +
          ":" +
          stack.slice(0, 256) +
          ":" +
          (cached.err || "").slice(0, 128) +
          ":" +
          getEffectiveGoVersion()
        );
      }
      return "exc:" + String(o.id) + ":" + String((cached.err || "").length) + ":" + (cached.err || "").slice(0, 128);
    }
    const cpuMs = typeof o.cpuMs === "number" ? o.cpuMs : o.t1 - o.t0;
    return (
      "plain:" +
      String(o.id) +
      ":" +
      String(o.t1 - o.t0) +
      ":" +
      String(cpuMs) +
      ":" +
      String(typeof o.wasmBytes === "number" ? o.wasmBytes : "") +
      ":" +
      String(typeof o.heapAlloc === "number" ? o.heapAlloc : "") +
      ":" +
      (o.name || "") +
      ":" +
      String((o.err || "").length) +
      ":" +
      String((o.stack || "").length)
    );
  }

  function fillDetailSideFromOp(o) {
    const titleEl = document.getElementById("detail-side-title");
    if (!titleEl) return;
    const cached = opDetailsById[String(o.id)];
    if (cached) {
      const tag =
        cached.event === "panic"
          ? "PANIC"
          : cached.event === "exception"
            ? "EXCEPTION"
            : "EVENT";
      titleEl.textContent = tag + " · " + (cached.name || "");
      const stack = (cached.stack || "").trim();
      if (stack) {
        const rawCombined = (cached.err ? cached.err + "\n\n" : "") + stack;
        populateDetailContent({
          mode: "stack",
          errText: cached.err || "",
          stackText: stack,
          rawCombined: rawCombined,
        });
      } else {
        populateDetailContent({
          mode: "plain",
          plainText: (cached.err ? cached.err + "\n\n" : "") + "(no stack captured)",
        });
      }
    } else {
      titleEl.textContent = "#" + o.id + " · " + (o.name || "");
      populateDetailContent({ mode: "plain", plainText: buildPlainEventDetailText(o) });
    }
    lastDetailSyncKey = buildDetailSyncKey(o);
  }

  function applyDetailPanelState() {
    if (logSplit) {
      logSplit.classList.toggle("detail-collapsed", !detailPanelVisible);
    }
    if (btnDetailToggle) {
      btnDetailToggle.setAttribute("aria-pressed", detailPanelVisible ? "true" : "false");
      btnDetailToggle.title = detailPanelVisible ? "Hide details panel" : "Show details panel";
    }
  }

  function updateLogRowSelectionClasses() {
    const opRows = logEl.querySelectorAll('.log-row[data-row-type="op"]');
    for (let i = 0; i < opRows.length; i++) {
      const row = opRows[i];
      const rid = row.getAttribute("data-op-id");
      row.classList.toggle(
        "log-row--selected",
        selectedOpId != null && rid != null && rid === String(selectedOpId)
      );
    }
    const logRows = logEl.querySelectorAll('.log-row[data-row-type="log"]');
    for (let j = 0; j < logRows.length; j++) {
      const row = logRows[j];
      const lid = row.getAttribute("data-log-id");
      row.classList.toggle(
        "log-row--selected",
        selectedLogId != null && lid != null && lid === String(selectedLogId)
      );
    }
  }

  function showDetailForOp(o) {
    if (!o) return;
    detailPanelVisible = true;
    applyDetailPanelState();
    selectedLogId = null;
    selectedOpId = o.id;
    fillDetailSideFromOp(o);
    updateLogRowSelectionClasses();
  }

  function syncDetailSideAfterRender() {
    if (selectedLogId != null) {
      const L = findLogById(String(selectedLogId));
      const titleEl = document.getElementById("detail-side-title");
      if (!titleEl) return;
      if (!L) {
        titleEl.textContent = "Log details";
        populateDetailContent({
          mode: "empty",
          message: "Selected log entry is no longer in the rolling buffer.",
        });
        return;
      }
      const logKey = buildDetailSyncKeyForLog(L);
      if (logKey && logKey === lastDetailSyncKey) {
        return;
      }
      fillDetailSideFromLog(L);
      return;
    }
    if (selectedOpId == null) return;
    const o = findOpById(String(selectedOpId));
    const titleEl = document.getElementById("detail-side-title");
    if (!titleEl) return;
    if (!o) {
      titleEl.textContent = "Event details";
      populateDetailContent({
        mode: "empty",
        message:
          "Selected event #" + selectedOpId + " is no longer in the rolling buffer.",
      });
      return;
    }
    const syncKey = buildDetailSyncKey(o);
    if (syncKey && syncKey === lastDetailSyncKey) {
      return;
    }
    fillDetailSideFromOp(o);
  }

  function wireDetailSidePanel() {
    if (btnDetailToggle) {
      btnDetailToggle.addEventListener("click", function () {
        detailPanelVisible = !detailPanelVisible;
        applyDetailPanelState();
      });
    }
    if (btnDetailHide) {
      btnDetailHide.addEventListener("click", function () {
        detailPanelVisible = false;
        applyDetailPanelState();
      });
    }
    document.addEventListener("keydown", function (ev) {
      if (ev.key !== "Escape") return;
      if (!detailPanelVisible) return;
      detailPanelVisible = false;
      applyDetailPanelState();
    });
  }

  logEl.addEventListener("click", function (ev) {
    const row = ev.target.closest(".log-row");
    if (!row) return;
    const rt = row.getAttribute("data-row-type");
    if (rt === "log") {
      const lid = row.getAttribute("data-log-id");
      if (!lid) return;
      const L = findLogById(lid);
      if (!L) return;
      showDetailForLog(L);
      return;
    }
    if (rt !== "op") return;
    const id = row.getAttribute("data-op-id");
    if (!id) return;
    const o = findOpById(id);
    if (!o) return;
    showDetailForOp(o);
  });

  wireDetailSidePanel();
  wireDetailTabs();
  applyDetailPanelState();

  function refreshEventLogPanel() {
    if (!logEl) return;
    logEl.textContent = "";
    logEl.appendChild(renderLog());
    syncDetailSideAfterRender();
    updateLogRowSelectionClasses();
  }

  if (logSearchInput) {
    logSearchInput.addEventListener("input", refreshEventLogPanel);
  }

  function render() {
    refreshEventDetailsMap();
    const bm = bucketOps();
    const keys = lastSeconds(36);
    cpuBars.textContent = "";
    ramBars.textContent = "";
    cpuBars.appendChild(renderBars(keys, bm, "cpu"));
    ramBars.appendChild(renderBars(keys, bm, "ram"));
    timelineEl.textContent = "";
    timelineEl.appendChild(renderSessionTimeline());
    logEl.textContent = "";
    logEl.appendChild(renderLog());
    syncDetailSideAfterRender();
    updateRamCapLabel();
    refreshStatusLine();
    updatePauseButtons();
  }

  document.getElementById("btn-pause").addEventListener("click", pauseLive);
  document.getElementById("btn-resume").addEventListener("click", resumeLive);

  statusEl.textContent = "waiting for app…";
  bc.postMessage({ type: "gu_sync" });
  startIntervals();
})();
