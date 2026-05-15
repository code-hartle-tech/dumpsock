// DumpSock v2 — vanilla JS frontend.
// Wails binds gui.App; calls go through window.go.gui.App.<Method>(...).

(function () {
  "use strict";

  const $ = (sel) => document.querySelector(sel);
  const $$ = (sel) => Array.from(document.querySelectorAll(sel));
  const body = document.body;

  const DONUT_CIRCUMFERENCE = 2 * Math.PI * 78; // matches r=78 in the SVG

  const state = {
    device: null,
    outputDir: "",
    appInfo: null,
    storage: null,
  };

  function setTab(tab) { body.dataset.tab = tab; }
  function toast(msg, ms = 4500) {
    const t = $("#toast");
    t.textContent = msg;
    t.hidden = false;
    clearTimeout(toast._t);
    toast._t = setTimeout(() => { t.hidden = true; }, ms);
  }

  function humanBytes(n) {
    if (!n && n !== 0) return "—";
    if (n < 1024) return n + " B";
    const units = ["KB", "MB", "GB", "TB"];
    let i = -1;
    do { n = n / 1024; i++; } while (n >= 1024 && i < units.length - 1);
    return n.toFixed(n >= 100 ? 0 : 1) + " " + units[i];
  }

  // ── bootstrap ────────────────────────────────────────────────────

  async function bootstrap() {
    try {
      state.appInfo = await window.go.gui.App.Info();
      $("#version").textContent = state.appInfo.version || "dev";
      $("#version-about").textContent = state.appInfo.version || "dev";
      $("#platform-about").textContent = state.appInfo.platform || "—";
    } catch (e) {
      console.warn("Info() unavailable yet:", e);
    }
    await rescan();
  }

  // ── device discovery ─────────────────────────────────────────────

  async function rescan() {
    let devices = [];
    try {
      devices = await window.go.gui.App.ListDevices();
    } catch (e) {
      toast("Couldn't reach usbmuxd. Is the macOS service running?");
      showDeviceEmpty();
      return;
    }
    const usb = devices.filter((d) => d.connection_type === "USB");
    const picked = usb[0] || devices[0] || null;
    if (!picked) {
      showDeviceEmpty();
      return;
    }
    state.device = picked;
    $("#device-name").textContent = picked.name || picked.udid;
    $("#device-spec").textContent =
      [picked.product_type, picked.product_version ? "iOS " + picked.product_version : "",
       picked.connection_type].filter(Boolean).join(" · ");
    $("#device-empty").hidden = true;
    $("#device-row").hidden = false;

    // Restore last-used output path.
    if (!state.outputDir) {
      try {
        const cfg = await window.go.gui.App.GetConfig();
        const last = cfg && (cfg.last_output || cfg.LastOutput);
        if (last) state.outputDir = last;
      } catch {}
      if (!state.outputDir) {
        try {
          state.outputDir = await window.go.gui.App.DefaultOutputFor(picked.name || "iPhone");
        } catch { state.outputDir = ""; }
      }
    }
    syncOutputPath();
    await refreshStorage();
  }

  function showDeviceEmpty() {
    $("#device-empty").hidden = false;
    $("#device-row").hidden = true;
    state.device = null;
    state.storage = null;
    renderGauge(null);
  }

  function syncOutputPath() {
    $("#output-path-display").textContent = state.outputDir || "—";
    $("#output-path").value = state.outputDir || "";
  }

  // ── storage gauge ────────────────────────────────────────────────

  async function refreshStorage() {
    if (!state.device) return;
    try {
      const s = await window.go.gui.App.DeviceStorage(state.device.udid || "");
      state.storage = s;
      renderGauge(s);
    } catch (e) {
      state.storage = null;
      renderGauge(null);
    }
  }

  function renderGauge(s) {
    const fillCircle = $("#donut-used");
    if (!s || !s.total_bytes) {
      $("#donut-value").textContent = "—";
      $("#donut-label").textContent = "no device";
      $("#gauge-state").textContent = "—";
      $("#storage-used").textContent = "—";
      $("#storage-free").textContent = "—";
      $("#storage-total").textContent = "—";
      $("#device-storage-pill").textContent = "—";
      fillCircle.setAttribute("stroke-dasharray", "0 " + DONUT_CIRCUMFERENCE);
      return;
    }
    const usedPct = s.used_bytes / s.total_bytes;
    const arc = DONUT_CIRCUMFERENCE * usedPct;
    fillCircle.setAttribute("stroke-dasharray", arc + " " + DONUT_CIRCUMFERENCE);
    $("#donut-value").textContent = humanBytes(s.used_bytes);
    $("#donut-label").textContent = "of " + humanBytes(s.total_bytes);
    $("#gauge-state").textContent = humanBytes(s.free_bytes) + " free";
    $("#storage-used").textContent = humanBytes(s.used_bytes) + " (" + Math.round(usedPct * 100) + "%)";
    $("#storage-free").textContent = humanBytes(s.free_bytes) + " (" + Math.round((1 - usedPct) * 100) + "%)";
    $("#storage-total").textContent = humanBytes(s.total_bytes);
    $("#device-storage-pill").textContent = humanBytes(s.free_bytes) + " free of " + humanBytes(s.total_bytes);
  }

  // ── output picker ────────────────────────────────────────────────

  async function pickOutput() {
    try {
      const dir = await window.go.gui.App.PickDirectory("Choose DumpSock output folder");
      if (dir) {
        state.outputDir = dir;
        syncOutputPath();
        try { await window.go.gui.App.SaveLastOutput(dir); } catch {}
      }
    } catch (e) {
      toast("Couldn't open the folder picker: " + (e.message || e));
    }
  }

  // ── pull flow ────────────────────────────────────────────────────

  function readPullForm(deleteConfirmed) {
    const wantsDelete = $("#delete-after").checked;
    return {
      udid: (state.device && state.device.udid) || "",
      output_root: state.outputDir,
      since: $("#since").value,
      until: $("#until").value,
      parallel: 4,
      until_found: 0,
      no_mtime: $("#no-mtime").checked,
      no_notify: $("#no-notify").checked,
      dry_run: $("#dry-run").checked,
      delete_after: wantsDelete && !!deleteConfirmed,
      confirm_delete: wantsDelete && !!deleteConfirmed,
    };
  }

  async function startPull() {
    const wantsDelete = $("#delete-after").checked;
    if (!state.device) { toast("Plug an iPhone in first."); return; }
    if (!state.outputDir) {
      toast("Pick an output folder in Settings first.");
      setTab("settings");
      return;
    }
    if (wantsDelete) {
      $("#delete-confirm").hidden = false;
      return;
    }
    await launchBackup(false);
  }

  async function launchBackup(deleteConfirmed) {
    const req = readPullForm(deleteConfirmed);
    resetProgressUI();
    setTab("progress");
    try {
      await window.go.gui.App.StartBackup(req);
    } catch (e) {
      toast("Couldn't start: " + (e.message || e));
      setTab("dashboard");
    }
  }

  function cancelPull() {
    window.go.gui.App.CancelBackup();
    $("#progress-phase").textContent = "Cancelling…";
  }

  function pct(done, total) {
    if (!total) return 0;
    return Math.max(0, Math.min(100, (done / total) * 100));
  }

  function fmtPhase(ev) {
    switch (ev.phase) {
      case "indexing": return "Indexing existing files…";
      case "walking":  return "Walking the device…";
      case "planning": return "Planning…";
      case "pulling":  return "Pulling files";
      case "deleting": return "Deleting from device…";
      case "done":     return "Done";
      case "error":    return "Error";
      default:         return ev.phase;
    }
  }

  function resetProgressUI() {
    $("#progress-phase").textContent = "Connecting…";
    $("#progress-fill").style.width = "0%";
    $("#count-done").textContent = "0";
    $("#count-total").textContent = "0";
    $("#count-rate").textContent = "";
    $("#current-file").textContent = "—";
    ["pulled", "skipped", "suffixed", "nodate", "errors"].forEach((k) => {
      $("#s-" + k).textContent = "0";
    });
    $("#log").textContent = "";
    $("#done-banner").hidden = true;
    $("#done-banner").classList.remove("error");
    $("#btn-cancel").hidden = false;
  }

  function onProgress(ev) {
    $("#progress-phase").textContent = fmtPhase(ev);
    if (ev.total) {
      $("#count-total").textContent = ev.total;
      $("#progress-fill").style.width = pct(ev.done || 0, ev.total) + "%";
    }
    if (typeof ev.done === "number") $("#count-done").textContent = ev.done;
    if (ev.current) $("#current-file").textContent = ev.current;
    if (typeof ev.pulled === "number") $("#s-pulled").textContent = ev.pulled;
    if (typeof ev.pre_skipped === "number" || typeof ev.post_skipped === "number") {
      $("#s-skipped").textContent = (ev.pre_skipped || 0) + (ev.post_skipped || 0);
    }
    if (typeof ev.suffixed === "number") $("#s-suffixed").textContent = ev.suffixed;
    if (typeof ev.nodate === "number") $("#s-nodate").textContent = ev.nodate;
    if (typeof ev.errors === "number") $("#s-errors").textContent = ev.errors;
  }

  function onLog(line) {
    const el = $("#log");
    el.textContent += line;
    el.scrollTop = el.scrollHeight;
  }

  function onDone(payload) {
    $("#btn-cancel").hidden = true;
    $("#done-banner").hidden = false;
    if (payload.error) {
      $("#progress-phase").textContent = "Stopped";
      $("#done-banner").classList.add("error");
      $("#done-summary").textContent = payload.error;
      $("#done-hint").hidden = true;
    } else {
      const r = payload.result || {};
      $("#progress-phase").textContent = "Done";
      $("#done-banner").classList.remove("error");
      $("#progress-fill").style.width = "100%";
      const parts = [];
      parts.push((r.Pulled || 0) + " pulled");
      const skipped = (r.PreSkipped || 0) + (r.PostSkipped || 0);
      if (skipped) parts.push(skipped + " skipped");
      if (r.Suffixed) parts.push(r.Suffixed + " suffixed");
      if (r.NoDate) parts.push(r.NoDate + " without a date");
      if (r.Filtered) parts.push(r.Filtered + " filtered");
      if (r.Deleted) parts.push(r.Deleted + " deleted from device");
      if (r.DeleteErrors) parts.push(r.DeleteErrors + " delete errors");
      if (r.Errors) parts.push(r.Errors + " errors");
      $("#done-summary").textContent = parts.join(" · ");
      $("#done-hint").hidden = !(r.Deleted > 0);
      // Refresh storage after the run — the iPhone's free space should have moved.
      refreshStorage();
    }
  }

  // ── bind ─────────────────────────────────────────────────────────

  function bindUI() {
    // Tab nav — sidebar items + any in-content link with data-tab-target
    $$("[data-tab-target]").forEach((b) => {
      b.addEventListener("click", (ev) => {
        if (b.tagName === "A") ev.preventDefault();
        setTab(b.dataset.tabTarget);
      });
    });

    // Dashboard
    $("#btn-rescan").addEventListener("click", rescan);
    $("#btn-pick-output").addEventListener("click", pickOutput);
    $("#btn-pull").addEventListener("click", startPull);

    // Last-backup card "View in Finder"
    const btnRevealOutput = $("#btn-reveal-output");
    if (btnRevealOutput) {
      btnRevealOutput.addEventListener("click", async () => {
        try { await window.go.gui.App.RevealInFinder(state.outputDir); }
        catch (e) { toast("Couldn't open the folder: " + (e.message || e)); }
      });
    }

    // Settings
    $("#btn-pick-output-settings").addEventListener("click", pickOutput);

    // Progress
    $("#btn-cancel").addEventListener("click", cancelPull);
    $("#btn-reveal").addEventListener("click", async () => {
      try { await window.go.gui.App.RevealInFinder(state.outputDir); }
      catch (e) { toast("Couldn't open the folder: " + (e.message || e)); }
    });
    $("#btn-again").addEventListener("click", () => setTab("dashboard"));

    // Logs
    const btnClearLog = $("#btn-clear-log");
    if (btnClearLog) {
      btnClearLog.addEventListener("click", () => {
        $("#log").textContent = "Log cleared.";
      });
    }

    // Delete confirmation modal
    $("#btn-cancel-delete").addEventListener("click", () => {
      $("#delete-confirm").hidden = true;
    });
    $("#btn-confirm-delete").addEventListener("click", async () => {
      $("#delete-confirm").hidden = true;
      await launchBackup(true);
    });
  }

  function bindRuntime() {
    if (!window.runtime || !window.runtime.EventsOn) {
      console.warn("Wails runtime not yet present");
      return;
    }
    window.runtime.EventsOn("backup:progress", onProgress);
    window.runtime.EventsOn("backup:log", onLog);
    window.runtime.EventsOn("backup:done", onDone);
  }

  function ready(fn) {
    if (document.readyState !== "loading") fn();
    else document.addEventListener("DOMContentLoaded", fn);
  }

  ready(() => {
    bindUI();
    bindRuntime();
    setTimeout(bootstrap, 50);
  });
})();
