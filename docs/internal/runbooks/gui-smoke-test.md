# Runbook — Hand-test the GUI

Mandatory before any release. Total time ~12 minutes.

## Setup

- iPhone with a non-trivial DCIM (~50+ photos, including 1+ Live Photo, 1+ video).
- USB-C cable known to do data (some lighting/USB-C cables are charge-only).
- DumpSock fresh build: `scripts/build-gui.sh`.
- Destination disk with >5 GB free.

## 1. First open

- [ ] Open `dist/DumpSock.app`. Window opens **maximized**.
- [ ] No element of the chrome bar is buried under macOS traffic lights.
- [ ] Sidebar shows: Dashboard, Compare & Merge, Backups, Logs, Settings + Help, About at bottom.
- [ ] No console errors. (Right-click → Inspect Element to verify.)

## 2. Empty state (no iPhone)

- [ ] With iPhone unplugged, dashboard shows an empty-device card with copy: "Plug in an iPhone or iPad to begin."
- [ ] No misleading storage stats are visible.

## 3. Plug iPhone in

- [ ] Within ~3 s, device card populates with name, model, iOS version, free + total bytes.
- [ ] Storage bar visualizes free vs total.
- [ ] Status dot is green and pulses softly.

## 4. Unplug iPhone

- [ ] Within ~3 s (next poll), device card empties back to the empty state.
- [ ] Storage stats clear; no stale numbers linger.
- [ ] Status dot disappears.

(This was issue #7 — the failure mode it fixes is exactly the regression to watch for.)

## 5. Plug back in, run dry-run backup

- [ ] Re-plug.
- [ ] Pick a temp destination via the folder picker.
- [ ] Toggle "Dry run".
- [ ] Click Start.
- [ ] Phase indicator walks through: indexing → walking → planning → done.
- [ ] Summary modal shows count + size that would be pulled. No files written to destination.
- [ ] Close modal, sidebar returns to Dashboard.

## 6. Real backup (small)

- [ ] In Settings, set `--until-found 0` (do everything).
- [ ] Pick a fresh temp destination.
- [ ] Click Start.
- [ ] Phase indicator walks through all phases.
- [ ] During `pulling`, per-file progress emits — bar is NOT stuck on "Planning".
- [ ] Done. Open destination in Finder via the Reveal button. Folders are `YYYY-MM-DD/` with photos inside.
- [ ] Re-open Finder reveal twice. Each open should land in Finder, not browser. (Regression check for the BrowserOpenURL bug.)

## 7. Re-run same backup

- [ ] Same destination, click Start again.
- [ ] Phase = `walking → planning → done` almost immediately.
- [ ] Summary shows `pre_skipped` ≈ count from previous run; `pulled` = 0.

## 8. Backup with delete-after

- [ ] Pick a destination on **a different folder** to avoid polluting your real backups.
- [ ] Enable "Wring it dry" (delete-after) in Settings.
- [ ] Confirm the modal warning.
- [ ] Run on 2-3 photos only (you can pre-filter with `--since` on a recent date).
- [ ] Backup completes, deletion phase runs.
- [ ] Open Photos.app on the iPhone. The deleted photos appear as ghosts with an exclamation mark (this is the AFC quirk; expected).
- [ ] iPhone Settings → Storage → free bytes goes up by the size of deleted photos within ~1 minute.
- [ ] Tap one of the ghost photos; Photos.app silently removes the orphan row. (You can also instruct the user to "Photos → Recently Deleted → empty" if any are there from manual deletions.)

## 9. Cancel

- [ ] Start a fresh backup, click Cancel mid-pull.
- [ ] Backup stops within ~2 s. Partial files at destination are kept; no zero-byte stubs.
- [ ] Phase = `canceled`. Summary modal shows partial counts.

## 10. Logs tab

- [ ] Switch to Logs. Recent events present.
- [ ] Click "Copy" — clipboard now contains the log text.
- [ ] Click "Clear" — log clears.

## 11. About / Help

- [ ] About modal: version + git SHA visible.
- [ ] Help opens an external link to public docs in the default browser.

## 12. Quit

- [ ] Cmd-Q quits cleanly.
- [ ] Re-open. Last output folder is restored. (Regression check for the config-load race.)

## What to do when a check fails

1. Capture screenshot, attach to a new GitHub issue.
2. Tag with `gui-regression`.
3. Block the release until fixed.

## What is NOT in this checklist

- Performance benchmarks (separate manual benchmark, see `scripts/bench.sh` once it exists).
- Multi-device disambiguation (we don't have a second test iPhone yet — issue #11).
- Windows / Linux runs (separate runbook page, tracked).
