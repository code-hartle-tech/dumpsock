// DumpSock v3 — vanilla JS frontend. Wails binds gui.App.

(function () {
  "use strict";

  const $  = (sel) => document.querySelector(sel);
  const $$ = (sel) => Array.from(document.querySelectorAll(sel));
  const body = document.body;

  const DONUT_R = 78;
  const DONUT_C = 2 * Math.PI * DONUT_R;

  // Rough category estimates of iPhone disk use — typical mix from
  // Apple's published "manage storage" guidance. Until we wire a proper
  // PhotoData / installed-app stats path, these proportions paint the
  // donut so it's not just a flat slab.
  const CATEGORY_MIX = {
    photos: 0.55,
    videos: 0.28,
    apps:   0.08,
    other:  0.09,
  };

  const state = {
    device: null,
    outputDir: "",
    appInfo: null,
    storage: null,
    jobs: JSON.parse(localStorage.getItem("dumpsock.jobs") || "[]"),
  };

  function setTab(tab) {
    // "backups" is an alias for "progress" — same underlying panel.
    body.dataset.tab = tab;
  }
  function toast(msg, ms = 4500) {
    const t = $("#toast");
    t.textContent = msg;
    t.hidden = false;
    clearTimeout(toast._t);
    toast._t = setTimeout(() => { t.hidden = true; }, ms);
  }
  function humanBytes(n) {
    if (n === undefined || n === null) return "—";
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
      const v = state.appInfo.version || "dev";
      $("#version").textContent = "v " + v;
      $("#version-about").textContent = v;
      $("#platform-about").textContent = state.appInfo.platform || "—";
    } catch {}
    renderJobHistory();
    await rescan();
  }

  // ── device ───────────────────────────────────────────────────────

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
    if (!picked) { showDeviceEmpty(); return; }

    state.device = picked;
    $("#device-name").textContent = picked.name || picked.udid;
    $("#device-spec").textContent =
      [picked.product_type, picked.product_version ? "iOS " + picked.product_version : "",
       picked.connection_type].filter(Boolean).join(" · ");
    $("#device-empty").hidden = true;
    $("#device-row").hidden = false;
    $("#compare-source-name").textContent = picked.name || picked.udid;

    if (!state.outputDir) {
      try {
        const cfg = await window.go.gui.App.GetConfig();
        const last = cfg && (cfg.last_output || cfg.LastOutput);
        if (last) state.outputDir = last;
      } catch {}
      if (!state.outputDir) {
        try { state.outputDir = await window.go.gui.App.DefaultOutputFor(picked.name || "iPhone"); }
        catch { state.outputDir = ""; }
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
    renderDonut(null);
  }
  function syncOutputPath() {
    $("#output-path-display").textContent = state.outputDir || "—";
    $("#output-path").value = state.outputDir || "";
    $("#compare-dest-name").textContent =
      state.outputDir ? state.outputDir.split("/").slice(-1)[0] || state.outputDir : "—";
  }

  // ── storage donut (multi-segment) ────────────────────────────────

  async function refreshStorage() {
    if (!state.device) return;
    try {
      const s = await window.go.gui.App.DeviceStorage(state.device.udid || "");
      state.storage = s;
      renderDonut(s);
    } catch {
      state.storage = null;
      renderDonut(null);
    }
  }

  function setSegment(id, arc, offsetArc) {
    const el = document.getElementById(id);
    if (!el) return;
    // dasharray = "<visible> <rest>"  — drawing arc length only
    el.setAttribute("stroke-dasharray", arc + " " + (DONUT_C - arc));
    el.setAttribute("stroke-dashoffset", -offsetArc);
  }

  function renderDonut(s) {
    if (!s || !s.total_bytes) {
      $("#donut-value").textContent = "—";
      $("#donut-label").textContent = "no device";
      $("#gauge-state").textContent = "—";
      ["photos","videos","apps","other"].forEach((k) => $("#legend-" + k).textContent = "—");
      $("#device-storage-pill").textContent = "—";
      ["seg-photos","seg-videos","seg-apps","seg-other"].forEach((id) => setSegment(id, 0, 0));
      return;
    }
    const usedPct = s.used_bytes / s.total_bytes;
    const totalArc = DONUT_C * usedPct;
    const photosArc = totalArc * CATEGORY_MIX.photos;
    const videosArc = totalArc * CATEGORY_MIX.videos;
    const appsArc   = totalArc * CATEGORY_MIX.apps;
    const otherArc  = totalArc * CATEGORY_MIX.other;
    setSegment("seg-photos", photosArc, 0);
    setSegment("seg-videos", videosArc, photosArc);
    setSegment("seg-apps",   appsArc,   photosArc + videosArc);
    setSegment("seg-other",  otherArc,  photosArc + videosArc + appsArc);

    $("#donut-value").textContent = humanBytes(s.used_bytes);
    $("#donut-label").textContent = "of " + humanBytes(s.total_bytes);
    $("#gauge-state").textContent = humanBytes(s.free_bytes) + " free";
    const fmt = (frac) => {
      const bytes = s.used_bytes * frac;
      return humanBytes(bytes) + " (" + Math.round(frac * 100) + "%)";
    };
    $("#legend-photos").textContent = fmt(CATEGORY_MIX.photos);
    $("#legend-videos").textContent = fmt(CATEGORY_MIX.videos);
    $("#legend-apps").textContent   = fmt(CATEGORY_MIX.apps);
    $("#legend-other").textContent  = fmt(CATEGORY_MIX.other);
    $("#device-storage-pill").textContent =
      humanBytes(s.free_bytes) + " free / " + humanBytes(s.total_bytes);
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
    if (wantsDelete) { $("#delete-confirm").hidden = false; return; }
    await launchBackup(false);
  }
  async function launchBackup(deleteConfirmed) {
    const req = readPullForm(deleteConfirmed);
    resetProgressUI();
    setTab("backups");
    try { await window.go.gui.App.StartBackup(req); }
    catch (e) {
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
    ["pulled","skipped","suffixed","nodate","errors"].forEach((k) => { $("#s-"+k).textContent = "0"; });
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
    if (typeof ev.pre_skipped === "number" || typeof ev.post_skipped === "number")
      $("#s-skipped").textContent = (ev.pre_skipped || 0) + (ev.post_skipped || 0);
    if (typeof ev.suffixed === "number") $("#s-suffixed").textContent = ev.suffixed;
    if (typeof ev.nodate === "number")  $("#s-nodate").textContent = ev.nodate;
    if (typeof ev.errors === "number")  $("#s-errors").textContent = ev.errors;
  }
  function onLog(line) {
    const el = $("#log");
    if (el.textContent.startsWith("No log yet")) el.textContent = "";
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
      recordJob({ status: "Failed", summary: payload.error });
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
      const success = !r.Errors && !r.DeleteErrors;
      const warned  = (r.Errors || 0) + (r.DeleteErrors || 0) > 0 && r.Pulled > 0;
      recordJob({
        status: success ? "Success" : (warned ? "Warning" : "Failed"),
        summary: parts.join(" · "),
        device: (state.device && state.device.name) || "iPhone",
        bytes: estimateJobBytes(r),
      });
      refreshStorage();
      // Update Last Backup card on the dashboard
      $("#last-backup-time").textContent = "Just now";
      $("#last-backup-status").hidden = false;
      $("#last-backup-detail").textContent = (state.outputDir || "—") + " · " + parts.join(" · ");
    }
  }
  function estimateJobBytes(r) {
    // Free space delta would be ideal; using pull count as a proxy is acceptable.
    return (r.Pulled || 0) * 1024 * 1024 * 8; // rough avg 8 MB per pulled file
  }

  // ── job history ──────────────────────────────────────────────────

  function recordJob(j) {
    j.t = Date.now();
    state.jobs.unshift(j);
    state.jobs = state.jobs.slice(0, 50);
    localStorage.setItem("dumpsock.jobs", JSON.stringify(state.jobs));
    renderJobHistory();
  }
  function renderJobHistory() {
    const tb = $("#jobs-table-body");
    if (!tb) return;
    if (state.jobs.length === 0) {
      tb.innerHTML = '<tr><td colspan="4" style="text-align:center;padding:32px;color:var(--dark-gray);">No backups yet — run one to start a history.</td></tr>';
      return;
    }
    tb.innerHTML = state.jobs.map((j) => {
      const dt = new Date(j.t).toLocaleString();
      const cls = j.status === "Success" ? "success" : j.status === "Warning" ? "warning" : "danger";
      return `<tr>
        <td><b>Backup</b> – ${j.device || "iPhone"}</td>
        <td><span class="pill ${cls}"><span class="dot"></span>${j.status}</span></td>
        <td>${dt}</td>
        <td class="size" style="text-align:right;">${humanBytes(j.bytes)}</td>
      </tr>`;
    }).join("");
  }

  // ── logs tab toggle ──────────────────────────────────────────────

  function bindLogTabs() {
    $$(".logs-tab").forEach((b) => {
      b.addEventListener("click", () => {
        $$(".logs-tab").forEach((x) => x.classList.toggle("active", x === b));
        const which = b.dataset.logsTab;
        $$("[data-logs-pane]").forEach((p) => { p.hidden = p.dataset.logsPane !== which; });
      });
    });
  }

  function bindFilterPills() {
    $$(".filter-pill").forEach((b) => {
      b.addEventListener("click", () => {
        $$(".filter-pill").forEach((x) => x.classList.toggle("active", x === b));
      });
    });
  }

  // ── bind ─────────────────────────────────────────────────────────

  function bindUI() {
    // Tab nav — sidebar items + any element with data-tab-target
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

    const btnRevealOutput = $("#btn-reveal-output");
    if (btnRevealOutput) {
      btnRevealOutput.addEventListener("click", async () => {
        try { await window.go.gui.App.RevealInFinder(state.outputDir); }
        catch (e) { toast("Couldn't open the folder: " + (e.message || e)); }
      });
    }

    // Settings
    $("#btn-pick-output-settings").addEventListener("click", pickOutput);

    // Chrome action goes to Settings
    const btnSettingsChrome = $("#btn-settings-chrome");
    if (btnSettingsChrome) btnSettingsChrome.addEventListener("click", () => setTab("settings"));

    // Backups / Progress
    $("#btn-cancel").addEventListener("click", cancelPull);
    $("#btn-reveal").addEventListener("click", async () => {
      try { await window.go.gui.App.RevealInFinder(state.outputDir); }
      catch (e) { toast("Couldn't open the folder: " + (e.message || e)); }
    });
    $("#btn-again").addEventListener("click", () => setTab("dashboard"));

    // Logs
    const btnClearLog = $("#btn-clear-log");
    if (btnClearLog) btnClearLog.addEventListener("click", () => { $("#log").textContent = "Log cleared."; });
    const btnCopyLog = $("#btn-copy-log");
    if (btnCopyLog) btnCopyLog.addEventListener("click", () => {
      const text = $("#log").textContent;
      navigator.clipboard.writeText(text).then(() => toast("Log copied."));
    });
    const btnExportLog = $("#btn-export-log");
    if (btnExportLog) btnExportLog.addEventListener("click", () => {
      const blob = new Blob([$("#log").textContent], { type: "text/plain" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "dumpsock-log-" + new Date().toISOString().replace(/[:.]/g, "-") + ".txt";
      a.click();
      URL.revokeObjectURL(url);
    });

    bindLogTabs();
    bindFilterPills();

    // Modal
    $("#btn-cancel-delete").addEventListener("click", () => { $("#delete-confirm").hidden = true; });
    $("#btn-confirm-delete").addEventListener("click", async () => {
      $("#delete-confirm").hidden = true;
      await launchBackup(true);
    });
  }
  function bindRuntime() {
    if (!window.runtime || !window.runtime.EventsOn) return;
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
