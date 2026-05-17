# SESSION_RESUME — DumpSock

> One-paragraph close-of-session snapshot, refreshed every session. Goal: next session rebuilds full state in under a minute.

**Last updated:** 2026-05-17

---

## Where we are

**Phase 0 — scaffold:** ✅ complete (commit `5bc2647`).
**Phase 1 — Go CLI feature parity + icloudpd extras:** ✅ merged.
**Phase 2 — Wails desktop GUI:** ✅ shipped (charcoal HARTLE.TECH theme; later rebranded blue per 2026-05-17 icon pick).
**Phase 2.5 — operator's 10-item feature sweep + nonnegotiables retrofit:** 🟢 **14 of ~21 tasks validated + committed at `dfbe600` (umbrella issue #27 on board #7). 6 in-flight awaiting operator test.**
**Phase 6/7 — full-device + iCloud backup:** 🛠 **scope expanded in CLAUDE.md; scaffold packages + CLI subcommands return ErrNotImplemented**. Real port is 2–10 weeks per agent estimate; roadmap at `docs/phase-6-roadmap.md`.

## What's on the working tree (uncommitted)

| Local Task | What changed | Status | Where |
|---|---|---|---|
| #1 | Gray out Password-protect when Compress unchecked | awaiting test | `main.js` syncEncryptState, `style.css` `.check.disabled` |
| #2 (item 4) | Top white chrome bar removed; sidebar runs full height | awaiting test | `index.html` (header removed), `style.css` (.chrome rules out), `cmd/dumpsock-gui/main.go` (unchanged — uses TitleBarHiddenInset) |
| #3 (item 8) | iPhone-row click no longer lands in DumpSock.app folder | awaiting test | `internal/gui/app.go::RevealDeviceInFinder` (drops bogus AppleScript, opens home), `main.js` (single toast) |
| #4 (item 1) | Byte-based progress bar + files-remaining counter | awaiting test | `internal/backup/backup.go` (`BytesTotal`/`BytesPulled`), `main.js::onProgress`, `style.css::.progress-files-line` |
| #5 (item 2) | Session restore via `.dumpsock-session.json` | awaiting test | `internal/backup/session.go`, `App.GetInterruptedSession`, frontend toast on launch |
| #6 (item 3) | Backups list + `.dumpsock.json` metadata + Move/Forget | awaiting test | `internal/backup/metadata.go`, `App.ListBackups/MoveBackup/ForgetBackup`, Backups tab card |
| #7 (item 7) | Dashboard pull-options: Wring it dry + Compress + Password-protect | awaiting test | `internal/backup/package.go` (zip + AES-256-GCM PBKDF2-SHA256), Dashboard `.pull-options`, password modal |
| #8 (item 5) | Browse iPhone tab with selective backup | awaiting test | `App.BrowseRemote`, `afc.Client.List/Stat`, `backup.Options.OnlyPaths`, Browse tab UI |
| #9 (item 9) | Storage breakdown clicks → Browse tab | awaiting test | `main.js` event delegation, `index.html` `data-browse-to` |
| #10 (item 6) | Third-party app data browser (UIFileSharingEnabled apps) | awaiting test | `App.ListFileSharingApps/BrowseApp` via go-ios `installationproxy`+`house_arrest`, Browse tab "App data" radio + dropdown |
| #11 (item 10) | Scope expansion: Phase 6/7 scaffolds + CLI subcommands + roadmap doc | awaiting sign-off | `internal/mb2/`, `internal/icloud/`, `internal/cli/{icloud,backupdevice}.go`, `docs/phase-6-roadmap.md`, CLAUDE.md non-goals softened |

**Build state right now:**

- `dist/DumpSock.app` — 12 MB, freshly rebuilt
- `dist/dumpsock-5b8ab9c-dirty-darwin-arm64` (and amd64/linux/windows variants) — fresh CLI binaries; `dumpsock icloud pull` + `dumpsock backup-device` both work, both return ErrNotImplemented with a pointer to the roadmap.

## Workflow rules established this session

See `~/.claude/projects/-Users-vz-Projects-dumpsock/memory/feedback_nonnegotiables.md` for the binding rules (commit-after-test, docs-with-code, tasks-before-work, project-board sync, memory-always-current). These are **nonnegotiable** going forward.

## What's still open

| # | Item | Status |
|---|---|---|
| 1 | Operator validates the 11 in-progress items end-to-end | ⏳ blocking commits |
| 2 | After operator pass: git commits per item (separate feature commits) | blocked on #1 |
| 3 | After operator pass: wiki + docs + landing-page updates reflecting new features | blocked on #1 |
| 4 | Existing GH issues #22 #23 #24 #25 → move to "In Progress" on org project #7 board; create new issues for items 1/2/3/4/6/7/8 | should happen alongside #1 |
| 5 | Phase 6 — real MobileBackup2 port (~3–5kLOC, 2–4 weeks for unencrypted MVP) | future |
| 6 | Phase 7 — iCloud Photos SRP-6a auth + CloudKit queries (~2–3 weeks port + maintenance tax) | future |
| 7 | Phase 8 — backup extract/mount (NOT restore — Apple-broken on iOS 17+) | future |
| 8 | Phase 4 — codesigning + notarization (SACRED #2: $99/yr + $200–500/yr — needs operator yes) | future |
| 9 | Phase 5 — mobile-to-mobile native Android | future |

## Next step on resume

1. **Operator tests each Task #1–#11.** Use `dist/DumpSock.app` + `dist/dumpsock-5b8ab9c-dirty-*`.
2. For each pass → TaskUpdate to completed; create the corresponding feature commit (`feat(scope): subject`); move the GH project-board card to Done.
3. For each fail → keep in_progress; fix, rebuild, retest.
4. Once the sweep is clean: update wiki + docs + landing.
5. Tag a `v0.x.y` (decision: bump minor for the feature sweep) and ship a draft GitHub Release.
6. Start Phase 6 implementation (MobileBackup2 port) per `docs/phase-6-roadmap.md`.

## What's running

- Nothing in the background.
- 10 background research agents from earlier in this session all completed. Their reports are in this session's transcript; the syntheses landed in `docs/phase-6-roadmap.md` (for items 6/8/10) and inline in the implementation choices for items 5/6.
