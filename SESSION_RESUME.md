# SESSION_RESUME — DumpSock

> One-paragraph close-of-session snapshot, refreshed every session. Goal: next session rebuilds full state in under a minute.

**Last updated:** 2026-05-17 (end of phase-2.5 sweep)

---

## Where we are

| Phase | Status |
|---|---|
| 0 — scaffold | ✅ `5bc2647` |
| 1 — Go CLI feature parity + icloudpd extras | ✅ merged |
| 2 — Wails desktop GUI | ✅ shipped, blue-bg icon (#66D1FF) adopted |
| **2.5 — operator's 10-item sweep + nonnegotiable retrofit** | 🟢 **17 of 22 tasks validated + committed (`dfbe600` + `bbfe540`). 5 in-flight, operator-blocked.** |
| 6 — MobileBackup2 backup engine | scaffolded (`internal/mb2/`, `dumpsock backup-device` CLI) |
| 7 — iCloud Photos download | scaffolded (`internal/icloud/`, `dumpsock icloud pull` CLI) |
| 8 — backup inspection & extraction | future (replaces the "restore-to-device" non-goal) |

## What's shipped this session (committed to develop, pushed)

`dfbe600` — feature sweep:

- Top white chrome bar removed; sidebar runs full window height.
- iPhone-row click no longer lands in DumpSock.app's folder.
- Byte-based progress bar + "N of M files copied · K remaining" line.
- Password-protect ↔ Compress UI dependency.
- Browse iPhone tab with one-level AFC view + selective backup (`OnlyPaths`).
- Storage donut click-through → Browse tab.
- Third-party app data browser via `BrowseUserApps` + `UIFileSharingEnabled` filter + in-repo VendDocuments wrapper.
- Folder selection + folder Delete (via `RemoveAll`) + folder backup (via `ExpandRemoteSelection`).
- `RemoteRemove` post-Stat verification ("iOS held onto this" surfaces honestly).
- In-app confirm modal (replaces `window.confirm()` which doesn't surface in Wails webview).
- Readable Delete button (`.btn-text-danger`, fixes red-on-red invisible label).
- AFC error codes humanised in English (PERM_DENIED, NOT_FOUND, NO_SPACE_LEFT, …).
- Greyed-out rows for iOS-hardened entries (🔒 icon + "hidden by iOS" + footer count).
- Scope expansion: CLAUDE.md non-goals softened; scaffolds for Phase 6/7; `docs/phase-6-roadmap.md` ships the plan.

`bbfe540` — docs sweep:

- External `features/browse-and-pick.md` + `features/encrypted-backups.md`.
- Landing-page feature cards refreshed.
- `reference/cli-flags.md` documents `dumpsock icloud pull` + `dumpsock backup-device` scaffolds.
- Internal wiki: two discoveries (`go-ios-house-arrest-hardcodes-vendcontainer`, `go-ios-browsefilesharingapps-doesnt-filter`) + session log (`2026-05-17-phase-2.5-sweep.md`).
- VitePress sidebar configs updated.

GH issues: #27 umbrella for the sweep; #28 upstream go-ios PR ask for AFC RENAME_PATH.

## What's in-flight (operator action required)

| Task | What to test |
|---|---|
| #5 Session restore | Crash a backup mid-flight (Ctrl-Q the app); relaunch and expect "your last run was interrupted (X/Y, ~N%)" toast. |
| #6 Backups list + Move/Forget | Backups tab → see saved-backups list. Try Move… to a different folder (cross-volume tests `copyTree` fallback). Try Forget. Try unplugging a volume and confirm the "plug Lexar back in" badge appears. |
| #7+#20 Compress + Password | Dashboard → enable Compress (and optionally Password). Run a small backup. Expect done-banner to show "zipped → X.zip" or "encrypted → X.zip.aes"; toast surfaces the full artifact path. |
| #11 Phase 6/7 scope expansion | Read `CLAUDE.md` non-goals section and `docs/phase-6-roadmap.md`. Confirm the framing (Phase 6/7 as opt-in subcommands, local-AFC remains the default pillar). |

## What's owed (deferred until in-flight tasks validate)

- Internal wiki tutorials for Browse + Compress/Password.
- Move-a-backup runbook.
- Crash-resume runbook.
- Built-in `dumpsock decrypt` companion CLI (format already shipped + Python snippet in docs).

## Workflow rules (binding, see memory)

1. Commit AFTER testing.
2. Update wiki / docs / landing on every state change.
3. TaskCreate BEFORE touching work.
4. Sync the GitHub project board.
5. Update memory + SESSION_RESUME after every delivery.

## Build commands

```bash
bash scripts/build-gui.sh        # macOS .app  (~30s)
bash scripts/build.sh            # CLI matrix  (~20s)
bash scripts/build-icons.sh      # icon regen  (~3s)
```

`gpg-agent` now uses `pinentry-mac` (installed this session) — future commits pop a GUI prompt instead of failing with `Inappropriate ioctl for device`.

## Next steps on resume

1. Operator validates #5, #6, #7+#20, signs off on #11.
2. Each pass → small per-validation commit if any code changed.
3. Sweep fully green → tag `v0.2.0` → draft GitHub Release.
4. Start Phase 6 (MobileBackup2 protocol port) per `docs/phase-6-roadmap.md`. Recommendation: unencrypted-first MVP (~2-4 focused weeks, ~3-5kLOC port from libimobiledevice).
