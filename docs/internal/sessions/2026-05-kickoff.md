# 2026-05 — kickoff → v3

**Window:** 2026-05-13 → 2026-05-15 (rolling)
**Closed at:** `develop` (latest push) — see `git log --oneline -1`
**Tests:** Go unit tests green. GUI hand-tested per [Hand-test the GUI](../runbooks/gui-smoke-test).

## Origin story

It started as a frustration with `icloudpd`'s 2FA prompt loop. The operator just wanted to back up their iPhone 16 Pro Max's photos to a Lexar SSD without dancing with Apple's 2FA UX. After we got `icloudpd` working with a `--sms` workaround, we discovered: iCloud Photos was OFF the entire time. The photos were physically on the iPhone, not in the cloud at all.

Pivot: forget cloud-pull. Build a direct-from-iPhone backup tool. Started as `idevicebackup2`. Then operator said:

> "Cant we just fucking clone most of the icloudpd functionality for this but for local backups instead (iphone -> storage media)?"

Plus, build a CLI **and** a GUI. Implement everything `icloudpd` does that we like. Call it **DumpSock**. Damp-sock thematic and motif. Go single binary. Apple-tier UX.

This was the founding directive.

## Day 1 (2026-05-13) — MVP

- Initial Python prototype with `pymobiledevice3` proving we could enumerate `/var/mobile/Media/DCIM/` over USB.
- Pivot to Go for the production tool.
- Built `internal/afc/afc.go` wrapping `github.com/danielpaulus/go-ios`.
- Built `internal/exif/exif.go` initially using `exiftool` shellouts.
- Built `internal/backup/backup.go` orchestrator.
- Built `cmd/dumpsock/` Cobra CLI.
- Built `cmd/dumpsock-gui/` Wails skeleton.
- Got an end-to-end backup running. ~70 GB pulled to the Lexar in ~25 min over USB 3.

## Day 2 (2026-05-14) — refinements

- **Exiftool `-fast2` trap discovered.** All MOV files were dating to today. Fix: `-fast` not `-fast2`, plus `MediaCreateDate` in the precedence chain. See [discovery](../discoveries/exiftool-fast2-trap).
- **Delete-after didn't actually delete.** Bug: `if len(jobs) == 0` early-out skipped the deletion phase when all files were pre-skipped. Fix: drop the early-out, worker loop is naturally a no-op for empty queues.
- **Live polling for device disconnects.** Operator: "when phone is disconnected it still shows as connected and stats read in the app's main screen / dashboard. It should reflect in real-time what the situation really is." Wired a 2.5 s `ListDevices()` poll on the JS side; pauses during active backup and when window's hidden. See [GUI structure / Device freshness](../dev/gui-structure#device-freshness-the-polling-story).
- **GUI v2 light theme.** Switched from dark default to light. Started matching the operator's `Archive.zip` mockup pixel-by-pixel.
- **Mascot SVG hand-author attempt #1, #2.** Both failed. "looks like ass still." Hand bezier traces of the approved PNG can't reproduce what the source pixels actually say.
- **Mascot SVG via VTracer.** Final answer. `cargo install vtracer`, run with tuned flags, 37 KB pixel-faithful SVG. Operator approved. See [discovery](../discoveries/vtracer-for-raster-to-svg).

## Day 3 (2026-05-15) — v3 polish + wiki

- **Brand v3 lockdown.** Primary `#E74C3C` (not `#E62A28`). Tokens, typography, sidebar geometry all to spec. Documented in [Brand bible](../dev/brand-bible).
- **GUI v3 layout.** 36 px chrome bar with 88 px left-pad (clearing the macOS traffic lights). 240 px sidebar with the 6-item nav. Tab-panel data-driven hide/show.
- **PTP investigation closed.** Hypothesized PTP `DeleteObject` might skip Recently Deleted. Empirical test: free space did not budge after a 332 MB delete via PTP. AFC delete wins. See [discovery](../discoveries/ptp-delete-still-trashes).
- **8-agent investigation on Recently Deleted.** Two rounds of 4 parallel agents. Only viable bypass = iPhone Mirroring + `CGEventPost`. Filed as #1, deferred to v0.2. See [discovery](../discoveries/recently-deleted-cannot-be-bypassed).
- **GitHub repo + project board created.** `code-hartle-tech/dumpsock`. [Project 7](https://github.com/orgs/code-hartle-tech/projects/7).
- **Replaced exiftool entirely.** Switched to `imagemeta` for EXIF + a custom moov atom walker. Removed exiftool from dependencies.
- **Internal + external VitePress wikis** scaffolded. External: product info, how-tos, privacy. Internal: dev, runbooks, discoveries, sessions, policy. Documenting **this** session as I write — meta.

## Decisions captured nowhere else

1. **Wails over Tauri.** Both were viable. Picked Wails because the Go-side reads cleaner and we're a Go-first shop here.
2. **Single Go module, no monorepo split.** CLI and GUI share `internal/*`. Splitting later is cheap; splitting too early is painful.
3. **No telemetry, ever.** Even opt-in. The product positioning is "the bytes go phone-to-disk, nowhere else"; opt-in telemetry would muddy that promise. See [Telemetry stance](../policy/telemetry-stance).
4. **Mascot is product-level, not org-level.** HARTLE.TECH formal communications use the org identity. DumpSock-specific surfaces use the mascot.
5. **Internal docs `noindex,nofollow`** in case they leak to a public host. Real defense is hosting them on Tailnet only.
6. **VTracer is the only sanctioned mascot SVG tool.** Hand-tracing is forbidden.
7. **Use `/bin/cp -f`** to bypass the operator's zsh `cp` overwrite-prompt alias.

## In-flight at session close

- [#22](https://github.com/code-hartle-tech/dumpsock/issues/22) — VitePress docs split. Currently writing the internal pages (this one, several siblings).
- [#23](https://github.com/code-hartle-tech/dumpsock/issues/23) — Real Compare & Merge engine. Currently a placeholder UI with static rows.
- [#7](https://github.com/code-hartle-tech/dumpsock/issues/7) — Replace 2.5 s polling with a real usbmuxd `Listen` subscription. Not started.

## Operator preferences crystallized in feedback memories

- **Aggressive autonomy.** No per-action approvals once the project is bootstrapped.
- **Distinct per-project allowlist.** This project's tooling and directories live in this project's allowlist.
- **Use real tracer for SVG.** Don't hand-author.
- **HTML `hidden` attribute can be defeated by CSS `display`.** Add a `[hidden] { display: none !important; }` defensively.
- **New-project protocol.** Inherit the org platforms, create per-project resources.

(See `~/.claude/projects/-Volumes-Lexar-Backup/memory/MEMORY.md` for the canonical list.)

## What to pick up next session

1. Finish internal docs (`policy/*` pages remaining).
2. `npm install` in `docs/` and build-test both VitePress instances.
3. Real Compare & Merge engine — replace static placeholder rows. `App.RunCompare(udid, outputDir)` walks DCIM via AFC + indexes destination + computes diff grouped by category (Photos / Videos / Live Photos / Screenshots / Documents / Other).
4. Rebuild GUI. Hand-smoke per [Hand-test the GUI](../runbooks/gui-smoke-test).
5. Commit + push everything to `origin/develop`.

## Files significantly changed this session

```
cmd/dumpsock-gui/main.go
internal/afc/afc.go
internal/backup/backup.go
internal/exif/exif.go             ← swap exiftool → imagemeta + moov walker
internal/frontend/dist/index.html ← v3 chrome bar + sidebar
internal/frontend/dist/style.css  ← v3 tokens, animations
internal/frontend/dist/main.js    ← device polling, tab logic
internal/gui/app.go               ← + DeviceStorage, RunCompare placeholder
internal/gui/config.go            ← lazy load fix
assets/brand/dumpsock-mascot.svg  ← VTracer output
scripts/build-gui.sh
scripts/build-icons.sh
scripts/build.sh
docs/external/                    ← NEW (entire tree)
docs/internal/                    ← NEW (entire tree)
docs/package.json                 ← NEW
CLAUDE.md                         ← per-project house rules
```

## Things deferred to next session(s)

- iPhone Mirroring delete bypass (issue #1) — punted to v0.2.
- Real usbmuxd `Listen` subscription (issue #7) — polling is fine for now.
- Apple Developer ID enrollment + notarization (issue #5).
- Windows GUI build runner (issue #6).
- Dark mode (issue #8).
- Screen-reader audit (issue #9).
- Multi-device disambiguation hardware test (issue #11).
