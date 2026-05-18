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
    // Browse-tab state: current AFC path + map of selected paths→size.
    // browseSelectedIsDir runs in parallel so we know which entries are
    // folders (sentinel size 0; real bytes come from the backend's
    // ExpandRemoteSelection walk at backup time).
    browsePath: "/",
    browseSelected: {},
    browseSelectedIsDir: {},
    browseSource: "device",   // "device" | "app"
    browseAppBundle: "",
  };

  function setTab(tab) {
    // "backups" is an alias for "progress" — same underlying panel.
    body.dataset.tab = tab;
    // Toggle the `hidden` attribute on each .tab panel. CSS has
    // `.tab[hidden] { display: none }`, so this drives panel visibility.
    document.querySelectorAll(".tab[data-tab]").forEach((panel) => {
      panel.hidden = panel.dataset.tab !== tab;
    });
    if (tab === "backups") {
      // Lazy-refresh: fetch the saved-backups list each time the user
      // opens the tab (cheap; ListBackups walks the config registry).
      refreshBackupsList();
    }
    if (tab === "browse") {
      // Open Browse at the device root so the user sees /DCIM, /Books, ...
      if (!state.browsePath) state.browsePath = "/";
      refreshBrowse();
    }
  }

  // ── browse tab ───────────────────────────────────────────────────
  // Render AFC's one-level-deep view of the iPhone filesystem. Selection
  // is per-file; folders are navigated by click. "Backup selected"
  // routes through StartBackup with only_paths set.
  async function refreshBrowse() {
    const dev = state.device;
    const host = $("#browse-list");
    const empty = $("#browse-empty");
    const breadcrumb = $("#browse-breadcrumb");
    if (!host) return;
    host.querySelectorAll(".browse-row").forEach((n) => n.remove());
    const pathLabel = state.browseSource === "app"
      ? `[${state.browseAppBundle || "—"}] ${state.browsePath || "/"}`
      : (state.browsePath || "/");
    breadcrumb.textContent = pathLabel;
    if (!dev) {
      empty.textContent = "Plug in an iPhone, then come back here.";
      empty.hidden = false;
      return;
    }
    if (state.browseSource === "app" && !state.browseAppBundle) {
      empty.textContent = "Pick an app from the list above.";
      empty.hidden = false;
      return;
    }
    empty.textContent = "Loading…";
    empty.hidden = false;
    let entries = [];
    try {
      if (state.browseSource === "app") {
        entries = await window.go.gui.App.BrowseApp(dev.udid, state.browseAppBundle, state.browsePath || "/");
      } else {
        entries = await window.go.gui.App.BrowseRemote(dev.udid, state.browsePath || "/");
      }
    } catch (e) {
      empty.textContent = "Couldn't list this folder: " + (e.message || e);
      return;
    }
    if (!entries.length) {
      empty.textContent = "This folder is empty.";
      return;
    }
    empty.hidden = true;
    let hiddenCount = 0;
    for (const e of entries) {
      const isSel = !!state.browseSelected[e.path];
      const row = document.createElement("div");
      const cls = ["browse-row"];
      if (e.is_dir) cls.push("dir");
      if (e.unreadable) cls.push("unreadable");
      row.className = cls.join(" ");
      // Folders are now selectable too (whole-folder backup). Unreadable
      // rows (iOS 15+ permission cliff — /PhotoData/Metadata, app
      // sandboxes without UIFileSharingEnabled, etc.) are still listed
      // so the user knows the folder isn't empty, but their checkbox +
      // actions are disabled.
      const icon = e.unreadable ? "🔒" : (e.is_dir ? "📁" : "📄");
      const sizeLabel = e.unreadable
        ? "hidden by iOS"
        : (e.is_dir ? "folder" : humanBytes(e.size));
      row.innerHTML = `
        <label class="browse-pick">
          <input type="checkbox" ${isSel ? "checked" : ""} ${e.unreadable ? "disabled" : ""}>
        </label>
        <div class="browse-name">${icon} ${escapeHTML(e.name)}</div>
        <div class="browse-size muted small">${sizeLabel}</div>
        <div class="browse-row-actions">
          <button class="btn ghost small" data-act="rename" title="Rename — coming soon (go-ios doesn't yet expose AFC rename)" disabled>Rename</button>
          <button class="btn ghost small btn-text-danger" data-act="delete" title="Delete from device" ${e.unreadable ? "disabled" : ""}>Delete</button>
        </div>
      `;
      if (e.unreadable) {
        hiddenCount++;
        // Tooltip so the user can hover to see why.
        row.title = "iOS 15+ blocks access to this entry over AFC. Likely an app sandbox without UIFileSharingEnabled, or a Photos-managed Library subtree.";
        host.appendChild(row);
        continue; // no checkbox / nav / delete handlers
      }
      // Folder-name click navigates in; checkbox + action buttons don't.
      if (e.is_dir) {
        row.querySelector(".browse-name").addEventListener("click", () => {
          state.browsePath = e.path;
          refreshBrowse();
        });
      }
      const cb = row.querySelector('input[type="checkbox"]');
      cb.addEventListener("change", () => {
        if (cb.checked) {
          // For folders we record sentinel size 0 — real byte counts
          // come from the backend ExpandRemoteSelection walk.
          state.browseSelected[e.path] = e.is_dir ? 0 : e.size;
          state.browseSelectedIsDir[e.path] = !!e.is_dir;
        } else {
          delete state.browseSelected[e.path];
          delete state.browseSelectedIsDir[e.path];
        }
        renderBrowseSummary();
      });
      row.querySelector('[data-act="delete"]').addEventListener("click", async () => {
        const ok = await confirmAction(
          e.is_dir ? "Delete this folder?" : "Delete this file?",
          (e.is_dir
            ? `"${e.name}" and EVERYTHING inside it will be removed from your iPhone over USB.`
            : `"${e.name}" will be removed from your iPhone over USB.`) +
          "\n\nThis cannot be undone — there is no recycle bin on iOS.",
          "Delete"
        );
        if (!ok) return;
        try {
          await window.go.gui.App.RemoteRemove(state.device.udid, e.path);
          delete state.browseSelected[e.path];
          delete state.browseSelectedIsDir[e.path];
          toast("Deleted.");
          refreshBrowse();
        } catch (err) {
          toast("Delete failed: " + (err.message || err), 8000);
        }
      });
      host.appendChild(row);
    }
    // Footer hint when iOS hid entries — so the user knows this dir
    // isn't truly empty and which subtree to skip going forward.
    if (hiddenCount > 0) {
      const note = document.createElement("div");
      note.className = "browse-row hidden-note";
      note.innerHTML = `<div class="muted small" style="grid-column: 1 / -1; padding: 6px 4px;">🔒 ${hiddenCount} entr${hiddenCount === 1 ? "y is" : "ies are"} hidden by iOS — accessing them over AFC requires Finder, jailbreak, or the app's own UI.</div>`;
      host.appendChild(note);
    }
    renderBrowseSummary();
  }
  function renderBrowseSummary() {
    const paths = Object.keys(state.browseSelected || {});
    const folderCount = paths.filter((p) => state.browseSelectedIsDir[p]).length;
    const fileCount   = paths.length - folderCount;
    const total = paths.reduce((acc, p) => acc + (state.browseSelected[p] || 0), 0);
    let summary;
    if (!paths.length) {
      summary = "No files selected.";
    } else {
      const parts = [];
      if (fileCount)   parts.push(`${fileCount} file${fileCount   === 1 ? "" : "s"}`);
      if (folderCount) parts.push(`${folderCount} folder${folderCount === 1 ? "" : "s"}`);
      // Bytes are file-only; folder sizes are resolved at backup time.
      summary = `${parts.join(" + ")} selected${total ? ` · ${humanBytes(total)} (files only — folders expand at backup time)` : ""}`;
    }
    $("#browse-summary").textContent = summary;
    $("#btn-browse-pull").disabled = !paths.length;
  }

  // Render the Backups tab's "Saved backups" list. Each row shows
  // device + path + last-completed + counts, plus action buttons:
  // Open in Finder, Move…, Forget.
  async function refreshBackupsList() {
    let entries = [];
    try { entries = await window.go.gui.App.ListBackups(); } catch (e) {
      console.warn("ListBackups failed:", e);
    }
    const host = $("#backups-list");
    const empty = $("#backups-list-empty");
    if (!host) return;
    host.querySelectorAll(".backup-row").forEach((n) => n.remove());
    if (!entries || !entries.length) {
      if (empty) empty.hidden = false;
      return;
    }
    if (empty) empty.hidden = true;
    for (const e of entries) {
      const m = e.metadata || {};
      const updated = m.updated_at ? new Date(m.updated_at) : null;
      const updatedLabel = updated ? updated.toLocaleString() : "—";
      const files = m.total_files || 0;
      const bytes = humanBytes(m.total_bytes || 0);
      const interruptedBadge = e.interrupted
        ? `<span class="pill warning"><span class="dot"></span>Interrupted</span>` : "";
      const missingBadge = !e.reachable
        ? `<span class="pill danger"><span class="dot"></span>Unavailable${e.volume ? ` · plug ${e.volume} back in` : ""}</span>`
        : "";
      // Detect encrypted archive inside this backup folder. Drives the
      // optional "Decrypt" button on the row.
      const archives = Array.isArray(e.archives) ? e.archives : [];
      const aesArchive = archives.find((p) => p.endsWith(".zip.aes")) || "";
      const archiveBadge = archives.length
        ? `<span class="pill info"><span class="dot"></span>${aesArchive ? "Encrypted archive present" : "Archive present"}</span>` : "";
      const row = document.createElement("div");
      row.className = "backup-row" + (e.reachable ? "" : " unreachable");
      row.innerHTML = `
        <div class="backup-row-main">
          <div class="backup-row-title">${escapeHTML(m.device_name || "iPhone")} ${interruptedBadge} ${missingBadge} ${archiveBadge}</div>
          <div class="backup-row-path">${escapeHTML(e.path)}</div>
          <div class="backup-row-meta muted small">
            ${files} files · ${bytes} · last updated ${updatedLabel}
          </div>
        </div>
        <div class="backup-row-actions">
          <button class="btn ghost small" data-act="reveal" ${e.reachable ? "" : "disabled"}>Reveal</button>
          ${aesArchive ? `<button class="btn ghost small" data-act="decrypt">Decrypt</button>` : ""}
          <button class="btn ghost small" data-act="move">Move…</button>
          <button class="btn ghost small" data-act="forget">Forget</button>
        </div>
      `;
      row.querySelector("[data-act=reveal]").addEventListener("click", async () => {
        try { await window.go.gui.App.RevealInFinder(e.path); }
        catch (err) { toast("Could not open: " + (err.message || err)); }
      });
      const decryptBtn = row.querySelector('[data-act="decrypt"]');
      if (decryptBtn && aesArchive) decryptBtn.addEventListener("click", () => decryptFlow(aesArchive));
      row.querySelector("[data-act=move]").addEventListener("click", async () => {
        try {
          const picked = await window.go.gui.App.PickOutputFolder("Move backup to…");
          if (!picked) return;
          // Append the backup's leaf folder name so we land at <picked>/<name>.
          const leaf = e.path.split("/").filter(Boolean).pop() || "DumpSock-backup";
          const dst = picked.replace(/\/$/, "") + "/" + leaf;
          toast("Moving…", 60000);
          await window.go.gui.App.MoveBackup(e.path, dst);
          toast("Moved to " + dst, 5000);
          refreshBackupsList();
        } catch (err) {
          toast("Move failed: " + (err.message || err));
        }
      });
      row.querySelector("[data-act=forget]").addEventListener("click", async () => {
        if (!confirm("Forget this backup from the list? (Files on disk are not touched.)")) return;
        try { await window.go.gui.App.ForgetBackup(e.path); refreshBackupsList(); }
        catch (err) { toast("Forget failed: " + (err.message || err)); }
      });
      host.appendChild(row);
    }
  }
  function escapeHTML(s) {
    if (s == null) return "";
    return String(s)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
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

  // rescan(): one-shot device probe. Updates UI based on current state.
  // Called on user "Rescan" click AND on the live poller below.
  async function rescan(silent) {
    let devices = [];
    try {
      devices = await window.go.gui.App.ListDevices();
    } catch (e) {
      if (!silent) toast("Couldn't reach usbmuxd. Is the macOS service running?");
      showDeviceEmpty();
      return;
    }
    // ONLY consider USB-connected devices. usbmuxd remembers Wi-Fi-paired
    // iPhones and keeps reporting them as connection_type="Network" even
    // after the cable is gone — picking one of those is the source of the
    // "device looks connected when it's not" bug. AFC over network is not
    // a path we support, so a Network-only device is functionally absent.
    const usb = devices.filter((d) => d.connection_type === "USB");
    const picked = usb[0] || null;
    if (!picked) {
      showDeviceEmpty();
      return;
    }

    // If the same UDID is still attached, only refresh storage (cheap).
    const sameDevice = state.device && state.device.udid === picked.udid;
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
    // Resume-after-crash check: if there's a .dumpsock-session.json
    // in the output dir, the previous backup didn't complete. Surface
    // the option to resume (which is just a re-run — dedup handles it).
    if (state.outputDir && !state.resumePromptShown) {
      state.resumePromptShown = true;
      try {
        const s = await window.go.gui.App.GetInterruptedSession(state.outputDir);
        if (s && s.total_jobs) {
          const pct = s.bytes_total ? Math.floor(100 * (s.bytes_pulled || 0) / s.bytes_total) : 0;
          toast(`Your last backup was interrupted (${s.done || 0}/${s.total_jobs} files, ~${pct}% done). Click "Dump it" to resume.`, 8000);
        }
      } catch {}
    }
    // Refresh storage on first connect and at most every 10s afterward —
    // AFC DeviceInfo is cheap but we don't need to flood it.
    const now = Date.now();
    if (!sameDevice || !state.storageLastAt || (now - state.storageLastAt) > 10000) {
      state.storageLastAt = now;
      await refreshStorage();
    }
  }

  // Live device-presence poll: re-checks every 2.5s while the app is
  // visible, so unplugging the iPhone clears the dashboard within a
  // few seconds even if the user never clicks "Rescan".
  let _devicePoll;
  function startDevicePolling() {
    if (_devicePoll) clearInterval(_devicePoll);
    _devicePoll = setInterval(() => {
      // Skip while a backup is running — let the engine own the AFC line.
      if (body.dataset.tab === "progress" || body.dataset.tab === "backups") {
        if (state.backupActive) return;
      }
      rescan(true /* silent — don't toast on transient failures */);
    }, 2500);
  }
  function stopDevicePolling() {
    if (_devicePoll) { clearInterval(_devicePoll); _devicePoll = null; }
  }

  function showDeviceEmpty() {
    $("#device-empty").hidden = false;
    $("#device-row").hidden = true;
    state.device = null;
    state.storage = null;
    state.storageLastAt = 0;
    renderDonut(null);
    $("#compare-source-name").textContent = "—";
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
    // Pull options all live in Settings now (2026-05-18 operator
    // preference). Dashboard reads from there.
    const wantsDelete = $("#delete-after").checked;
    const wantsCompress = $("#dash-compress").checked;
    const wantsEncrypt = $("#dash-encrypt").checked;
    // Password mode: which radio is selected under the encrypt checkbox.
    let passwordMode = "";
    if (wantsEncrypt) {
      const checked = document.querySelector('input[name="password-mode"]:checked');
      passwordMode = checked ? checked.value : "standard";
    }
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
      // Encryption implies compression — we encrypt the produced zip.
      compress: wantsCompress || wantsEncrypt,
      password: wantsEncrypt ? (state.backupPassword || "") : "",
      password_mode: passwordMode,
      archive_scope: (document.querySelector('input[name="archive-scope"]:checked')?.value) || "all",
    };
  }
  async function startPull() {
    const wantsDelete = $("#delete-after").checked;
    const wantsEncrypt = $("#dash-encrypt").checked;
    if (!state.device) { toast("Plug an iPhone in first."); return; }
    if (!(await ensureOutputDir())) return;
    if (wantsEncrypt) {
      const useBio = $("#use-biometric") && $("#use-biometric").checked;
      let pw = null;
      // If Touch ID is enrolled, fetch the saved password via biometric
      // unlock — skips the password modal entirely.
      if (useBio && state.biometricEnrolled) {
        try {
          pw = await window.go.gui.App.LoadBiometricPassword();
        } catch (e) {
          toast("Touch ID unlock failed (" + (e.message || e) + ") — falling back to typed password.", 5000);
          pw = null;
        }
      }
      if (pw === null) {
        // Either Touch ID not enrolled / not asked / failed → prompt.
        pw = await promptPassword();
        if (pw === null) return; // user cancelled
        // If the user wants to enroll, stash the freshly-typed password
        // in Keychain AFTER the backup runs (don't risk enrolling a
        // password the user mistyped before they actually use it).
        state.enrollAfterBackup = useBio && !state.biometricEnrolled;
      }
      state.backupPassword = pw;
    } else {
      state.backupPassword = "";
      state.enrollAfterBackup = false;
    }
    if (wantsDelete) { $("#delete-confirm").hidden = false; return; }
    await launchBackup(false);
  }

  // Decrypt flow: takes an optional preselected path. If srcPath is
  // empty, opens the native file picker. Then prompts for a password
  // (single-field; we're DECRYPTING, no confirm needed), calls the
  // backend, and toasts the produced path with a Reveal-in-Finder
  // follow-up button. End-to-end in the GUI — operator preference
  // 2026-05-18 ("everything stays in the UI, don't be lazy").
  async function decryptFlow(srcPath) {
    let src = srcPath || "";
    if (!src) {
      try { src = await window.go.gui.App.PickArchiveToDecrypt(); }
      catch (e) { toast("File picker failed: " + (e.message || e)); return; }
      if (!src) return; // user cancelled
    }
    const pw = await promptDecryptPassword(src);
    if (pw === null) return; // user cancelled
    toast("Decrypting " + src.split("/").pop() + "…", 60000);
    try {
      const outPath = await window.go.gui.App.DecryptArchive(src, "", pw);
      toast("Decrypted to " + outPath + " — click here to reveal.", 12000);
      // Set the lastArchivePath state so the existing Reveal-archive
      // button on the done banner can also surface this file.
      state.lastArchivePath = outPath;
      try { await window.go.gui.App.RevealFileInFinder(outPath); } catch {}
    } catch (e) {
      toast("Decrypt failed: " + (e.message || e), 10000);
    }
  }

  // Single-password modal — reuses the existing #password-modal sheet
  // but hides the confirm field since we're DECRYPTING (no risk of
  // a typo silently locking the user out — wrong password just errors).
  function promptDecryptPassword(srcLabel) {
    return new Promise((resolve) => {
      const veil = $("#password-modal");
      const inp1 = $("#password-input");
      const inp2 = $("#password-confirm");
      const err  = $("#password-error");
      const btnOk = $("#btn-password-ok");
      const btnCancel = $("#btn-password-cancel");
      const title = veil.querySelector("h2");
      const subtitle = veil.querySelector("p");
      const confirmLabel = veil.querySelector('label[for="password-confirm"]');
      // Rewrite the modal copy for decrypt mode.
      const oldTitle = title.textContent;
      const oldSubtitle = subtitle.textContent;
      title.textContent = "Password for " + (srcLabel ? srcLabel.split("/").pop() : "this archive");
      subtitle.textContent = "Enter the password that was used when this archive was created.";
      inp2.parentNode.style.display = "none";
      confirmLabel.style.display = "none";
      inp1.value = ""; inp2.value = ""; err.textContent = "";
      veil.hidden = false;
      setTimeout(() => inp1.focus(), 50);
      const cleanup = (val) => {
        veil.hidden = true;
        btnOk.onclick = null;
        btnCancel.onclick = null;
        // Restore the modal to encrypt-mode copy for next time.
        title.textContent = oldTitle;
        subtitle.textContent = oldSubtitle;
        inp2.parentNode.style.display = "";
        confirmLabel.style.display = "";
        resolve(val);
      };
      btnOk.onclick = () => {
        const a = inp1.value;
        if (!a) { err.textContent = "Password required."; return; }
        cleanup(a);
      };
      btnCancel.onclick = () => cleanup(null);
    });
  }

  // In-app confirm modal. Substitute for window.confirm() — the
  // native confirm dialog doesn't surface reliably inside the Wails
  // webview on macOS (operator reported "delete never works" 2026-05-17;
  // the confirm() was silently returning falsy). Returns a Promise that
  // resolves to true (OK) or false (Cancel).
  function confirmAction(title, body, okLabel) {
    return new Promise((resolve) => {
      const veil = $("#confirm-modal");
      const btnOk = $("#btn-confirm-ok");
      const btnCancel = $("#btn-confirm-cancel");
      $("#confirm-title").textContent = title || "Are you sure?";
      $("#confirm-body").textContent  = body  || "";
      if (okLabel) btnOk.textContent = okLabel; else btnOk.textContent = "Confirm";
      veil.hidden = false;
      const cleanup = (val) => {
        veil.hidden = true;
        btnOk.onclick = null;
        btnCancel.onclick = null;
        resolve(val);
      };
      btnOk.onclick = () => cleanup(true);
      btnCancel.onclick = () => cleanup(false);
    });
  }

  // Inline password prompt — two matching entries required. Returns the
  // password string on success, null on cancel.
  function promptPassword() {
    return new Promise((resolve) => {
      const veil = $("#password-modal");
      const inp1 = $("#password-input");
      const inp2 = $("#password-confirm");
      const err  = $("#password-error");
      const btnOk = $("#btn-password-ok");
      const btnCancel = $("#btn-password-cancel");
      inp1.value = ""; inp2.value = ""; err.textContent = "";
      veil.hidden = false;
      setTimeout(() => inp1.focus(), 50);
      const cleanup = (val) => {
        veil.hidden = true;
        btnOk.onclick = null;
        btnCancel.onclick = null;
        resolve(val);
      };
      btnOk.onclick = () => {
        const a = inp1.value, b = inp2.value;
        if (!a) { err.textContent = "Password required."; return; }
        if (a.length < 8) { err.textContent = "Use at least 8 characters."; return; }
        if (a !== b) { err.textContent = "The two passwords don't match."; return; }
        cleanup(a);
      };
      btnCancel.onclick = () => cleanup(null);
    });
  }

  // revealOutputDir() opens the destination in Finder. If the path is no
  // longer accessible (drive unplugged, folder deleted), it routes through
  // ensureOutputDir() instead of barking a raw stat error toast.
  async function revealOutputDir() {
    if (!state.outputDir) {
      // No path yet — treat the action as "pick where to back up".
      await ensureOutputDir();
      return;
    }
    let exists = false;
    try { exists = await window.go.gui.App.PathExists(state.outputDir); }
    catch {}
    if (!exists) {
      toast("That backup folder isn't reachable — pick a new one.");
      if (!(await ensureOutputDir())) return;
    }
    try { await window.go.gui.App.RevealInFinder(state.outputDir); }
    catch (e) { toast("Couldn't open the folder: " + (e.message || e)); }
  }

  // ensureOutputDir() resolves the backup destination right before a run.
  // If no path is set, OR the saved path doesn't exist on disk (drive
  // unplugged, folder deleted, moved off-volume), open the native picker
  // so the user can choose a fresh location. Returns true if state.outputDir
  // is usable when it returns, false if the user cancelled.
  async function ensureOutputDir() {
    let p = state.outputDir;
    let exists = false;
    if (p) {
      try { exists = await window.go.gui.App.PathExists(p); }
      catch { exists = false; }
    }
    if (exists) return true;

    toast(p
      ? "Saved backup folder isn't there anymore — pick a new one."
      : "Pick where to save the backup."
    );
    try {
      const dir = await window.go.gui.App.PickDirectory("Choose backup folder");
      if (!dir) return false;
      state.outputDir = dir;
      syncOutputPath();
      try { await window.go.gui.App.SaveLastOutput(dir); } catch {}
      return true;
    } catch (e) {
      toast("Couldn't open the folder picker: " + (e.message || e));
      return false;
    }
  }
  async function launchBackup(deleteConfirmed) {
    const req = readPullForm(deleteConfirmed);
    resetProgressUI();
    setTab("backups");
    state.backupActive = true;
    try { await window.go.gui.App.StartBackup(req); }
    catch (e) {
      toast("Couldn't start: " + (e.message || e));
      state.backupActive = false;
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
      case "indexing":   return "Indexing existing files…";
      case "walking":    return "Walking the device…";
      case "planning":   return "Planning…";
      case "pulling":    return "Pulling files";
      case "deleting":   return "Deleting from device…";
      case "packaging":  return "Bundling into .zip…";
      case "encrypting": return "Encrypting archive…";
      case "done":       return "Done";
      case "error":      return "Error";
      default:           return ev.phase;
    }
  }
  // Human-readable byte count. Matches the rest of the GUI's unit display.
  function fmtBytes(n) {
    if (!Number.isFinite(n) || n <= 0) return "0 B";
    const u = ["B", "KB", "MB", "GB", "TB"];
    let i = 0;
    while (n >= 1024 && i < u.length - 1) { n /= 1024; i++; }
    return (i === 0 ? n.toString() : n.toFixed(n >= 100 ? 0 : n >= 10 ? 1 : 2)) + " " + u[i];
  }
  function resetProgressUI() {
    $("#progress-phase").textContent = "Connecting…";
    $("#progress-fill").style.width = "0%";
    $("#count-bytes-done").textContent = "0 B";
    $("#count-bytes-total").textContent = "0 B";
    $("#count-done").textContent = "0";
    $("#count-total").textContent = "0";
    $("#count-remaining").textContent = "0";
    $("#count-rate").textContent = "";
    $("#current-file").textContent = "—";
    ["pulled","skipped","suffixed","nodate","errors"].forEach((k) => { $("#s-"+k).textContent = "0"; });
    $("#done-banner").hidden = true;
    $("#done-banner").classList.remove("error");
    $("#btn-cancel").hidden = false;
  }
  function onProgress(ev) {
    $("#progress-phase").textContent = fmtPhase(ev);
    // Progress bar tracks BYTES (more honest than file count when files
    // are wildly different sizes — a 1.6 GB MOV among 800 thumbnails).
    // Fall back to file count if the engine hasn't emitted bytes yet.
    if (typeof ev.bytes_total === "number" && ev.bytes_total > 0) {
      $("#progress-fill").style.width = pct(ev.bytes_pulled || 0, ev.bytes_total) + "%";
      $("#count-bytes-done").textContent  = fmtBytes(ev.bytes_pulled || 0);
      $("#count-bytes-total").textContent = fmtBytes(ev.bytes_total);
    } else if (ev.total) {
      $("#progress-fill").style.width = pct(ev.done || 0, ev.total) + "%";
    }
    if (ev.total) {
      $("#count-total").textContent = ev.total;
      const done = ev.done || 0;
      $("#count-remaining").textContent = Math.max(0, ev.total - done);
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
    state.backupActive = false;
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
      // Compression / encryption status (task #7 / #20). The backend
      // attaches `zip_path`, `encrypted_path`, and/or `package_error`
      // to the done payload. We extend the summary so the user can SEE
      // that the .zip or .zip.aes was actually produced (and where).
      if (payload.package_error) {
        parts.push("packaging failed (" + payload.package_error + ")");
      } else if (payload.encrypted_path) {
        parts.push("encrypted → " + payload.encrypted_path.split("/").slice(-1)[0]);
      } else if (payload.zip_path) {
        parts.push("zipped → " + payload.zip_path.split("/").slice(-1)[0]);
      }
      $("#done-summary").textContent = parts.join(" · ");
      $("#done-hint").hidden = !(r.Deleted > 0);
      const success = !r.Errors && !r.DeleteErrors && !payload.package_error;
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
      // Prefer showing the packaged artifact path when one exists so the
      // user can find the .zip/.zip.aes file. Falls back to outputDir.
      $("#last-backup-detail").textContent =
        payload.encrypted_path || payload.zip_path || state.outputDir || "—";
      // Reveal the "View in Finder" button only once a real backup has run.
      const revealBtn = $("#btn-reveal-output");
      if (revealBtn) revealBtn.hidden = false;
      // If the user opted into Touch ID and this is the first backup
      // since enrollment, stash the password they just used. We only
      // do this on successful packaging — no point persisting a
      // password for an archive that didn't ship.
      if (state.enrollAfterBackup && state.backupPassword && !payload.package_error) {
        window.go.gui.App.EnrollBiometricPassword(state.backupPassword)
          .then(() => {
            state.biometricEnrolled = true;
            state.enrollAfterBackup = false;
            const statusEl = $("#biometric-status");
            if (statusEl) statusEl.textContent = "Enrolled. Future backups prompt Touch ID instead of asking for the password.";
            toast("Touch ID enrolled — next backup will skip the password prompt.", 6000);
          })
          .catch((e) => {
            const msg = (e.message || e).toString();
            // The unsigned-binary case ("errSecMissingEntitlement" or
            // SecAccessControl returning NULL) is by far the most
            // common reason this fails in dev. Surface it concretely.
            if (msg.includes("errSecMissingEntitlement") || msg.includes("SecAccessControlCreateWithFlags returned NULL")) {
              toast("Touch ID needs a code-signed DumpSock build — coming in Phase 4 (notarized release). Your password worked for this backup; just type it next time.", 9000);
              // Untick the checkbox so the user doesn't keep retrying.
              const cb = $("#use-biometric");
              if (cb) cb.checked = false;
              state.enrollAfterBackup = false;
            } else {
              toast("Couldn't enroll Touch ID: " + msg, 8000);
            }
          });
      }
      // Toast the artifact path explicitly — the done-banner can be
      // missed if the user has already switched tabs.
      const archivePath = payload.encrypted_path || payload.zip_path || "";
      if (archivePath) {
        // Stash so the Reveal Archive button (below) knows what to open.
        state.lastArchivePath = archivePath;
        const btnRevealArchive = $("#btn-reveal-archive");
        if (btnRevealArchive) btnRevealArchive.hidden = false;
        toast((payload.encrypted_path ? "Encrypted archive at: " : "Archive at: ") + archivePath +
              " — click 'Reveal archive' in the done banner to open in Finder.", 10000);
      }
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
        const f = b.dataset.filter;
        // The "Run Compare" pill is a button, not a filter; route it.
        if (f === "run") {
          runCompare();
          return;
        }
        $$(".filter-pill").forEach((x) => {
          if (x.dataset.filter === "run") return;
          x.classList.toggle("active", x === b);
        });
        applyCompareFilter(f);
      });
    });
  }

  // ── Compare & Merge ──────────────────────────────────────────────

  // Visibility filter on the per-side category rows. "all" shows all; the
  // delta filters dim rows whose category has zero relevant items.
  function applyCompareFilter(filter) {
    const last = state.compare;
    $$(".compare-row[data-cmp-cat]").forEach((row) => {
      const side = row.dataset.cmpSide;
      const cat  = row.dataset.cmpCat;
      let visible = true;
      if (last && filter !== "all") {
        // Only show rows on the side that has items in the chosen delta.
        // (Crude but matches the user's mental model: "show me what's new on
        // the device" → highlight device rows that have non-zero items.)
        if (filter === "device" && side !== "device") visible = false;
        if (filter === "backup" && side !== "backup") visible = false;
        if (filter === "different") {
          // "Different" applies to both sides — show only rows that
          // exist on both sides with non-zero counts.
          const devItems   = last.device  && last.device[cat]  && last.device[cat].items;
          const backItems  = last.backup  && last.backup[cat]  && last.backup[cat].items;
          if (!(devItems && backItems)) visible = false;
        }
      }
      row.style.display = visible ? "" : "none";
    });
  }

  function compareStatus(text, busy) {
    const el = $("#compare-status");
    if (!el) return;
    if (!text) { el.hidden = true; el.textContent = ""; return; }
    el.hidden = false;
    el.textContent = text;
    el.classList.toggle("busy", !!busy);
  }

  function fmtPhaseCompare(phase) {
    switch (phase) {
      case "walking_device": return "Walking iPhone…";
      case "walking_backup": return "Indexing backup folder…";
      case "categorizing":   return "Categorizing…";
      case "diffing":        return "Diffing names + sizes…";
      case "done":           return "";
      default:               return phase || "";
    }
  }

  async function runCompare() {
    if (!state.device) {
      toast("Plug an iPhone in first, then click Run Compare.");
      return;
    }
    if (!state.outputDir) {
      toast("Pick a destination folder in Settings, then re-run Compare.");
      setTab("settings");
      return;
    }
    const btn = $("#btn-compare-run");
    if (btn) btn.disabled = true;
    compareStatus("Walking iPhone…", true);
    const t0 = performance.now();
    try {
      const res = await window.go.gui.App.RunCompare(
        state.device.udid || "",
        state.outputDir,
      );
      const ms = Math.round(performance.now() - t0);
      state.compare = res;
      renderCompare(res, ms);
      compareStatus("");
    } catch (e) {
      compareStatus("Compare failed.", false);
      toast("Compare failed: " + (e.message || e));
    } finally {
      if (btn) btn.disabled = false;
    }
  }

  function setSideTotals(meta, total) {
    if (!meta) return;
    meta.textContent = total.items + " items · " + humanBytes(total.bytes);
  }
  function setRow(side, cat, side2) {
    const row = document.querySelector('.compare-row[data-cmp-side="' + side + '"][data-cmp-cat="' + cat + '"]');
    if (!row) return;
    const detail = row.querySelector(".compare-detail");
    if (!side2 || !side2.items) {
      detail.textContent = "0 items";
      row.style.opacity = 0.5;
      return;
    }
    detail.textContent = side2.items + " items · " + humanBytes(side2.bytes);
    row.style.opacity = 1;
  }

  function renderCompare(res, ms) {
    if (!res) return;
    const cats = ["photos","videos","live_photos","screenshots","documents","other"];
    cats.forEach((c) => {
      setRow("device", c, res.device  && res.device[c]);
      setRow("backup", c, res.backup  && res.backup[c]);
    });
    setSideTotals($("#compare-device-meta"), res.device_total || {items:0,bytes:0});
    setSideTotals($("#compare-backup-meta"), res.backup_total || {items:0,bytes:0});

    $("#cmp-stat-new-device").textContent = res.new_in_device ?? 0;
    $("#cmp-stat-new-backup").textContent = res.new_in_backup ?? 0;
    $("#cmp-stat-diff").textContent       = res.different ?? 0;
    $("#cmp-stat-elapsed").textContent    = ms != null ? (ms < 1000 ? ms + " ms" : (ms/1000).toFixed(1) + " s") : "—";

    const totalNew = (res.new_in_device ?? 0) + (res.new_in_backup ?? 0) + (res.different ?? 0);
    $("#cmp-count-all").textContent      = "(" + totalNew + ")";
    $("#cmp-count-device").textContent   = "(" + (res.new_in_device ?? 0) + ")";
    $("#cmp-count-backup").textContent   = "(" + (res.new_in_backup ?? 0) + ")";
    $("#cmp-count-diff").textContent     = "(" + (res.different ?? 0) + ")";

    const note = $("#compare-footnote");
    if (note) {
      note.innerHTML =
        '<strong>' + (res.new_in_device ?? 0) + '</strong> on the iPhone aren\'t at your backup yet · ' +
        '<strong>' + (res.new_in_backup ?? 0) + '</strong> at the backup aren\'t (or no longer are) on the iPhone · ' +
        '<strong>' + (res.different ?? 0) + '</strong> have the same name but different bytes.';
    }
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
      btnRevealOutput.addEventListener("click", async (ev) => {
        // Card-level handler also catches the click; stop bubble to avoid
        // running both handlers and double-opening Finder.
        ev.stopPropagation();
        await revealOutputDir();
      });
    }

    // Settings
    $("#btn-pick-output-settings").addEventListener("click", pickOutput);

    // Chrome action goes to Settings
    const btnSettingsChrome = $("#btn-settings-chrome");
    if (btnSettingsChrome) btnSettingsChrome.addEventListener("click", () => setTab("settings"));

    // Backups / Progress
    $("#btn-cancel").addEventListener("click", cancelPull);
    $("#btn-reveal").addEventListener("click", revealOutputDir);
    $("#btn-again").addEventListener("click", () => setTab("dashboard"));
    // Reveal the .zip / .zip.aes archive in Finder (selected, not just
    // its parent folder). Wired after the operator hit "where the heck
    // do I find iphone.zip.aes" on 2026-05-18.
    const btnRevealArchive = $("#btn-reveal-archive");
    if (btnRevealArchive) btnRevealArchive.addEventListener("click", async () => {
      const p = state.lastArchivePath;
      if (!p) { toast("No archive produced for the latest backup."); return; }
      try { await window.go.gui.App.RevealFileInFinder(p); }
      catch (e) { toast("Could not reveal: " + (e.message || e)); }
    });

    // Settings → Archive after backup: Password-protect is meaningless
    // without a .zip to encrypt (engine encrypts the produced archive,
    // not a folder tree), so it stays disabled until Bundle is ticked.
    // The Standard/Maximum radio is shown only when Password-protect
    // is checked.
    const compressCb = $("#dash-compress");
    const encryptCb  = $("#dash-encrypt");
    const modeRow    = $("#password-mode-row");
    const biometricRow = $("#biometric-row");
    const useBiometricCb = $("#use-biometric");
    const biometricStatus = $("#biometric-status");
    function syncEncryptState() {
      if (!compressCb || !encryptCb) return;
      const enabled = compressCb.checked;
      encryptCb.disabled = !enabled;
      const label = encryptCb.closest("label");
      if (label) label.classList.toggle("disabled", !enabled);
      if (!enabled) encryptCb.checked = false;
      if (modeRow) modeRow.hidden = !encryptCb.checked;
    }
    if (compressCb) compressCb.addEventListener("change", syncEncryptState);
    if (encryptCb) encryptCb.addEventListener("change", syncEncryptState);
    syncEncryptState();

    // Touch ID enrollment status. Probe once on app start so the
    // Settings row appears (or stays hidden on Linux/Windows). The
    // checkbox reflects whether a password is already enrolled; the
    // status line below adapts to the current state.
    (async function initBiometric() {
      if (!biometricRow || !useBiometricCb) return;
      let available = false;
      try { available = await window.go.gui.App.BiometricsAvailable(); } catch {}
      if (!available) {
        // Linux/Windows or Mac without login password → leave hidden.
        biometricRow.hidden = true;
        return;
      }
      biometricRow.hidden = false;
      let enrolled = false;
      try { enrolled = await window.go.gui.App.HasBiometricPassword(); } catch {}
      useBiometricCb.checked = !!enrolled;
      state.biometricEnrolled = !!enrolled;
      renderBiometricStatus();
    })();

    function renderBiometricStatus() {
      if (!biometricStatus) return;
      biometricStatus.textContent = state.biometricEnrolled
        ? "Enrolled. Future backups prompt Touch ID instead of asking for the password."
        : "Not enrolled. Tick this box and run a backup — your password gets stored in Keychain afterward.";
    }

    if (useBiometricCb) useBiometricCb.addEventListener("change", async () => {
      // Unchecking → clear the stored password immediately.
      if (!useBiometricCb.checked && state.biometricEnrolled) {
        try {
          await window.go.gui.App.ClearBiometricPassword();
          state.biometricEnrolled = false;
          renderBiometricStatus();
          toast("Touch ID password cleared from Keychain.");
        } catch (e) {
          toast("Could not clear Keychain entry: " + (e.message || e));
        }
      }
      // Checking the box does NOT immediately enroll — we wait until the
      // user types their password (next backup) and stash it then.
      renderBiometricStatus();
    });

    // Backups list
    const btnRefreshBackups = $("#btn-refresh-backups");
    if (btnRefreshBackups) btnRefreshBackups.addEventListener("click", refreshBackupsList);

    // Decrypt-an-archive flow: pick file → prompt password → decrypt
    // → toast the produced path with a Reveal-in-Finder follow-up.
    const btnDecryptArchive = $("#btn-decrypt-archive");
    if (btnDecryptArchive) btnDecryptArchive.addEventListener("click", () => decryptFlow(""));

    // Storage breakdown click-through → Browse tab (item 9). Each legend
    // row carries a data-browse-to AFC path; clicking it navigates the
    // browser there. We use event delegation so segments added later
    // (e.g. dynamic re-render) also get the handler.
    document.body.addEventListener("click", (ev) => {
      const row = ev.target.closest("[data-browse-to]");
      if (!row) return;
      if (!state.device) { toast("Plug an iPhone in first."); return; }
      state.browsePath = row.dataset.browseTo || "/";
      setTab("browse");
    });

    // Donut segments themselves are clickable too — point all media
    // segments at /DCIM (apps/other → root). Same delegation pattern.
    const seg2path = { "seg-photos": "/DCIM", "seg-videos": "/DCIM", "seg-apps": "/", "seg-other": "/" };
    Object.keys(seg2path).forEach((id) => {
      const el = document.getElementById(id);
      if (!el) return;
      el.style.cursor = "pointer";
      el.addEventListener("click", () => {
        if (!state.device) { toast("Plug an iPhone in first."); return; }
        state.browsePath = seg2path[id];
        setTab("browse");
      });
    });

    // Browse source switcher (device vs app)
    document.querySelectorAll('input[name="browse-source"]').forEach((r) => {
      r.addEventListener("change", async () => {
        state.browseSource = r.value;
        state.browsePath = "/";
        state.browseSelected = {};
        renderBrowseSummary();
        const picker = $("#browse-app-picker");
        if (state.browseSource === "app") {
          picker.hidden = false;
          // Populate picker with file-sharing apps (lazy, on first use).
          if (state.device && picker.options.length <= 1) {
            picker.innerHTML = '<option value="">Loading apps…</option>';
            try {
              const apps = await window.go.gui.App.ListFileSharingApps(state.device.udid);
              picker.innerHTML = '<option value="">Choose an app…</option>' +
                apps.map((ap) =>
                  `<option value="${ap.bundle_id}">${escapeHTML(ap.name || ap.bundle_id)} (${ap.bundle_id})</option>`
                ).join("");
            } catch (e) {
              picker.innerHTML = `<option value="">${escapeHTML("Error: " + (e.message || e))}</option>`;
            }
          }
        } else {
          picker.hidden = true;
          refreshBrowse();
        }
      });
    });
    const browseAppPicker = $("#browse-app-picker");
    if (browseAppPicker) browseAppPicker.addEventListener("change", () => {
      state.browseAppBundle = browseAppPicker.value;
      state.browsePath = "/";
      state.browseSelected = {};
      refreshBrowse();
    });

    // Browse iPhone tab
    const btnBrowseUp = $("#browse-up");
    if (btnBrowseUp) btnBrowseUp.addEventListener("click", () => {
      const p = state.browsePath || "/";
      if (p === "/" || p === "") return;
      const parts = p.split("/").filter(Boolean);
      parts.pop();
      state.browsePath = parts.length ? "/" + parts.join("/") : "/";
      refreshBrowse();
    });
    const btnBrowseRefresh = $("#browse-refresh");
    if (btnBrowseRefresh) btnBrowseRefresh.addEventListener("click", refreshBrowse);
    const btnBrowsePull = $("#btn-browse-pull");
    if (btnBrowsePull) btnBrowsePull.addEventListener("click", async () => {
      const selected = Object.keys(state.browseSelected);
      if (!selected.length) { toast("Select at least one file or folder first."); return; }
      if (!state.device) { toast("Plug an iPhone in first."); return; }
      if (!(await ensureOutputDir())) return;
      const hasFolder = selected.some((p) => state.browseSelectedIsDir[p]);
      let flatPaths = selected;
      if (hasFolder) {
        // Expand folders to their full file list via the backend so the
        // engine's OnlyPaths receives plain file paths.
        toast("Expanding folder selection…", 60000);
        try {
          const entries = await window.go.gui.App.ExpandRemoteSelection(state.device.udid, selected);
          flatPaths = entries.map((e) => e.path);
          toast(`Expanded to ${flatPaths.length} file${flatPaths.length === 1 ? "" : "s"}.`, 3000);
        } catch (err) {
          toast("Could not expand folder selection: " + (err.message || err));
          return;
        }
        if (!flatPaths.length) { toast("Folder selection expanded to zero files."); return; }
      }
      const form = readPullForm(false);
      form.only_paths = flatPaths;
      setTab("backups");
      try {
        await window.go.gui.App.StartBackup(form);
        state.backupActive = true;
        resetProgressUI();
      } catch (e) {
        toast("Couldn't start: " + (e.message || e));
      }
    });

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

    // Compare & Merge — explicit Run button (also triggered by the pill).
    const btnCmpRun = $("#btn-compare-run");
    if (btnCmpRun) btnCmpRun.addEventListener("click", runCompare);
    const btnPullMissing = $("#btn-compare-pull-missing");
    if (btnPullMissing) btnPullMissing.addEventListener("click", () => {
      // A normal backup naturally pulls only the "new in device" set
      // (dedup index pre-skips matched files). One-click shortcut.
      if (!state.device || !state.outputDir) { toast("Hook up a device + destination first."); return; }
      startPull();
    });

    // Run Compare automatically the first time the user lands on the tab.
    $$("[data-tab-target=compare]").forEach((b) => {
      b.addEventListener("click", () => {
        if (!state.compare && state.device && state.outputDir && !state._compareAuto) {
          state._compareAuto = true;
          // Small delay so the tab is visible before the (potentially long) walk starts.
          setTimeout(runCompare, 120);
        }
      });
    });

    // Device row click → open Finder at the home folder and tell the
    // user where to find their iPhone. We don't try to auto-select the
    // iPhone in Finder's sidebar (Finder's AppleScript dictionary has no
    // `sidebar` property; System Events GUI-scripting works but would
    // require an Accessibility TCC prompt).
    const deviceRow = $("#device-row");
    if (deviceRow) deviceRow.addEventListener("click", async () => {
      const dev = state.device || {};
      try {
        await window.go.gui.App.RevealDeviceInFinder(dev.name || "");
        toast("Opened Finder. Your iPhone is in the sidebar under Locations — click it there to manage.", 5500);
      } catch (e) {
        toast("Couldn't open Finder: " + (e.message || e));
      }
    });

    // Last-backup card click → open the destination in Finder, with
    // graceful fallback to the picker if the drive isn't mounted.
    const lastBackupCard = $("#last-backup-card");
    if (lastBackupCard) lastBackupCard.addEventListener("click", async (ev) => {
      if (ev.target.closest("button")) return;
      await revealOutputDir();
    });

    // Sidebar "Help & Docs" anchor (target="_blank") — Wails' webview
    // doesn't open new browser windows for anchors. Intercept and shell
    // out to the system browser via runtime.BrowserOpenURL.
    document.body.addEventListener("click", (ev) => {
      const a = ev.target && ev.target.closest && ev.target.closest("a[href]");
      if (!a) return;
      const href = a.getAttribute("href");
      if (!href || href === "#" || href.startsWith("#")) return;
      if (/^(https?:|mailto:)/i.test(href)) {
        ev.preventDefault();
        if (window.runtime && window.runtime.BrowserOpenURL) {
          window.runtime.BrowserOpenURL(href);
        }
      }
    });
  }
  function bindRuntime() {
    if (!window.runtime || !window.runtime.EventsOn) return;
    window.runtime.EventsOn("backup:progress", onProgress);
    window.runtime.EventsOn("backup:log", onLog);
    window.runtime.EventsOn("backup:done", onDone);
    window.runtime.EventsOn("compare:progress", (p) => {
      compareStatus(fmtPhaseCompare(p && p.phase), p && p.phase !== "done");
    });
  }
  function ready(fn) {
    if (document.readyState !== "loading") fn();
    else document.addEventListener("DOMContentLoaded", fn);
  }
  // Pause polling when the window goes invisible (saves cycles + AFC traffic).
  document.addEventListener("visibilitychange", () => {
    if (document.hidden) stopDevicePolling();
    else { rescan(true); startDevicePolling(); }
  });

  ready(() => {
    bindUI();
    bindRuntime();
    setTimeout(() => {
      bootstrap().then(startDevicePolling);
    }, 50);
  });
})();
