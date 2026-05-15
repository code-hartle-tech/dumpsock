# 8 agents on "Recently Deleted"

## TL;DR

**Recently Deleted cannot be bypassed over USB.** The only viable workaround we found is **iPhone Mirroring + `CGEventPost`** — Apple substitutes Mac Touch ID for the iPhone Face ID prompt that gates "Delete from Recently Deleted" during a mirroring session. Filed as issue #1, deferred to v0.2.

## What we wanted

User runs DumpSock with `--delete-after`. We pull the photos to their Lexar. We delete them from the iPhone. Free space should go up by the size of what we pulled, **immediately and permanently** — not "in 30 days when Recently Deleted expires".

## What we got the first time

After deleting 332 MB worth of photos via AFC `Remove`:

- `/var/mobile/Media/DCIM/` shrinks by 332 MB.
- Photos.app shows ghosts (exclamation marks) for each deleted photo.
- iPhone Settings → Storage shows **free space unchanged**.

We assumed the bytes went to Recently Deleted (the trash-can equivalent in Photos.app). Wrong: AFC deletes bypass Photos.app entirely. The bytes are gone from DCIM but **Photos.sqlite still references them**, so Photos.app's "Recently Deleted" count is wrong AND the iPhone's storage stats are wrong because they read from PhotoData/Photos.sqlite, not from DCIM.

## What we tried (round 1: 4 agents)

| Agent | Hypothesis | Result |
|---|---|---|
| #1 | PTP DeleteObject (USB image protocol) routes through PhotoKit | False — routes through PhotoKit AND through Recently Deleted. Worse. |
| #2 | A second AFC service `com.apple.PhotoKit` exists | Doesn't exist on iOS 13+. |
| #3 | The Apple `DeviceCheck` framework would let an authed app trigger PhotoKit operations over USB | No. DeviceCheck is for app-attestation, not file ops. |
| #4 | An undocumented lockdownd service handles Photos.sqlite vacuum | Could not find one in `lockdownd`'s service list dump. |

After round 1, conclusion: **the bytes-vs-database mismatch is the real bug**, not "Recently Deleted is intercepting our deletes".

## What we tried (round 2: 4 more agents)

| Agent | Hypothesis | Result |
|---|---|---|
| #5 | Direct Photos.sqlite write over AFC | AFC sandboxes us out of `PhotoData/`. Can't reach the sqlite. |
| #6 | Trigger a Photos.app refresh via a URL scheme over usbmuxd | `photos-redirect://` works on-device only. |
| #7 | Use the Apple Configurator command set (`cfgutil`) to push a vacuum command | `cfgutil`'s commands don't include any Photos.app op. |
| #8 | **iPhone Mirroring (macOS 15+) with synthetic events** | ✅ Works. Mirroring substitutes Mac Touch ID for the on-device Face ID gate. We can scripted-drive "Recently Deleted → Select All → Delete → Confirm" via `CGEventPost`. |

## The viable bypass (issue #1)

Apple released **iPhone Mirroring** in macOS 15. Once mirroring is active, the iPhone surface appears as a window on the Mac. **Inside this session**, prompts that would normally require Face ID on the iPhone are confirmed via the Mac's Touch ID instead. This includes "Delete from Recently Deleted".

The bypass procedure:

1. Start iPhone Mirroring (`open -a "iPhone Mirroring"`).
2. Wait for the session window to appear.
3. Inject keyboard events via `CGEventPost` to navigate: Photos → Albums → Recently Deleted → Select → Delete All → Confirm.
4. The confirm step triggers a system prompt that the user satisfies with Touch ID once.
5. Recently Deleted now drains. Free space goes up.

We have a prototype script. It works on the operator's iPhone 16 Pro Max + iOS 26.4.2 + macOS 26.x.

## Why we didn't ship it

- **Fragile.** Apple can break Mirroring auto-confirm in any update. We'd need integration tests every release, against real hardware.
- **Accessibility entitlement required.** `CGEventPost` triggering on another app needs the user to grant DumpSock the Accessibility permission. Bad first-run UX.
- **Apple-API-adjacent gray area.** Apple might frown on shipping it. Better to wait for clarity (or for them to provide an actual API).
- **The current AFC-delete behavior is good enough.** Bytes are freed; the only cost is the Photos.app ghosts, which the user can clear in 10 seconds.

Deferred to v0.2. Tracked at [issue #1](https://github.com/code-hartle-tech/dumpsock/issues/1).

## What we ship today (v0.1)

- `--delete-after` does an AFC `Remove` per pulled file. Bytes are freed.
- The GUI warns the user that Photos.app may show ghosts for ~24h or until tapped.
- We document the manual cleanup in [public docs / Free up storage](../../external/guide/free-storage#after-delete-after-tidy-photos-app).

## Code

- `internal/afc/afc.go:Remove()` — the AFC-side delete.
- `internal/backup/backup.go:runDeletions()` — the worker loop that calls it.
- `internal/gui/app.go` exposes nothing related to Mirroring (deliberate; v0.2 territory).
