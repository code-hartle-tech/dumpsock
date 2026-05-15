// DumpSock GUI — vanilla JS. Wails binds gui.App; we call its methods
// via window.go.gui.App.<Method>(...).

(function () {
  "use strict";

  const $ = (sel) => document.querySelector(sel);
  const body = document.body;
  const state = {
    device: null,
    outputDir: "",
    appInfo: null,
  };

  function show(screen) { body.dataset.screen = screen; }

  function toast(msg, ms = 4500) {
    const t = $("#toast");
    t.textContent = msg;
    t.hidden = false;
    clearTimeout(toast._t);
    toast._t = setTimeout(() => { t.hidden = true; }, ms);
  }

  async function bootstrap() {
    try {
      state.appInfo = await window.go.gui.App.Info();
      $("#version").textContent = state.appInfo.version || "dev";
      $("#tagline").textContent = state.appInfo.tagline || "";
    } catch (e) {
      console.warn("Info() unavailable yet:", e);
    }
    await rescan();
  }

  async function rescan() {
    let devices = [];
    try {
      devices = await window.go.gui.App.ListDevices();
    } catch (e) {
      toast("Couldn't reach usbmuxd. Is the macOS service running?");
      show("empty");
      return;
    }
    // Prefer USB-connected.
    const usb = devices.filter((d) => d.connection_type === "USB");
    const picked = usb[0] || devices[0] || null;
    if (!picked) {
      show("empty");
      return;
    }
    state.device = picked;
    $("#device-name").textContent = picked.name || picked.udid;
    $("#device-spec").textContent =
      [picked.product_type, picked.product_version ? "iOS " + picked.product_version : "",
       picked.connection_type].filter(Boolean).join(" · ");

    if (!state.outputDir) {
      // Restore last-used path first; only fall back to the per-device
      // default on a brand-new install. Defensive on field name: most
      // Wails versions honor json tags but some old paths exposed
      // PascalCase, so try both.
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
      $("#output-path").value = state.outputDir;
    }
    show("config");
  }

  async function pickOutput() {
    try {
      const dir = await window.go.gui.App.PickDirectory("Choose DumpSock output folder");
      if (dir) {
        state.outputDir = dir;
        $("#output-path").value = dir;
        // Persist so next launch restores this folder.
        try {
          await window.go.gui.App.SaveLastOutput(dir);
        } catch (e) {
          console.warn("SaveLastOutput failed:", e);
        }
      }
    } catch (e) {
      toast("Couldn't open the folder picker: " + (e.message || e));
    }
  }

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
      // delete_after is enabled only after the user clicked through
      // the danger modal. The Go layer rejects delete_after without
      // confirm_delete; UI sets them together.
      delete_after: wantsDelete && !!deleteConfirmed,
      confirm_delete: wantsDelete && !!deleteConfirmed,
    };
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

  async function startPull() {
    const wantsDelete = $("#delete-after").checked;
    if (!state.outputDir) {
      toast("Pick an output folder first.");
      return;
    }
    if (wantsDelete) {
      // Defer the actual launch until the user confirms in the modal.
      $("#delete-confirm").hidden = false;
      return;
    }
    await launchBackup(false);
  }

  async function launchBackup(deleteConfirmed) {
    const req = readPullForm(deleteConfirmed);
    resetProgressUI();
    show("pulling");
    try {
      await window.go.gui.App.StartBackup(req);
    } catch (e) {
      toast("Couldn't start: " + (e.message || e));
      show("config");
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

  function onProgress(ev) {
    $("#progress-phase").textContent = fmtPhase(ev);
    if (ev.total) {
      $("#count-total").textContent = ev.total;
      $("#progress-fill").style.width = pct(ev.done || 0, ev.total) + "%";
    }
    if (ev.done) $("#count-done").textContent = ev.done;
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
    // Stay on the pulling screen; morph it into a done state in place.
    $("#btn-cancel").hidden = true;
    $("#done-banner").hidden = false;
    if (payload.error) {
      $("#progress-phase").textContent = "Stopped";
      $("#done-banner").classList.add("error");
      $("#done-summary").textContent = payload.error;
    } else {
      const r = payload.result || {};
      $("#progress-phase").textContent = "Done";
      $("#done-banner").classList.remove("error");
      // Fill the progress bar; rate/current become final.
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
    }
  }

  function bindUI() {
    $("#btn-rescan").addEventListener("click", rescan);
    $("#btn-rescan-empty").addEventListener("click", rescan);
    $("#btn-pick-output").addEventListener("click", pickOutput);
    $("#btn-pull").addEventListener("click", startPull);
    $("#btn-cancel").addEventListener("click", cancelPull);
    $("#btn-reveal").addEventListener("click", async () => {
      try {
        await window.go.gui.App.RevealInFinder(state.outputDir);
      } catch (e) {
        toast("Couldn't open the folder: " + (e.message || e));
      }
    });
    $("#btn-again").addEventListener("click", () => show("config"));

    // Delete-confirmation modal
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

  // Wails injects its runtime + bindings before DOMContentLoaded usually,
  // but be defensive.
  function ready(fn) {
    if (document.readyState !== "loading") fn();
    else document.addEventListener("DOMContentLoaded", fn);
  }

  ready(() => {
    bindUI();
    bindRuntime();
    // Tiny delay so the runtime has a chance to populate window.go.
    setTimeout(bootstrap, 50);
  });
})();
