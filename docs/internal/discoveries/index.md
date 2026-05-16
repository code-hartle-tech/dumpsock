# Discoveries

Things we learned the hard way. Each entry should answer:

1. **What was the surprise?**
2. **What did we test / try?**
3. **What's the canonical fix or workaround?**
4. **Where in the code does this live now?**

The format is loose. The point is that future-us (or a Claude session resuming cold) doesn't have to re-derive any of this.

## Catalog

| Page | One-liner |
|---|---|
| [8 agents on Recently Deleted](./recently-deleted-cannot-be-bypassed) | Across 8 parallel research agents, the only viable USB-side bypass of "Recently Deleted" is iPhone Mirroring + CGEventPost. Filed as #1. |
| [exiftool `-fast2` trap](./exiftool-fast2-trap) | `-fast2` skips the moov atom on MOV/MP4, silently defaulting capture date to today. Use `-fast`, not `-fast2`. We later replaced exiftool entirely with imagemeta + our own moov walker. |
| [PTP delete still trashes](./ptp-delete-still-trashes) | We hoped USB PTP `DeleteObject` (opcode `0x100B`) might skip Recently Deleted. iOS 26 routes it through PhotoKit anyway. Free space did not budge after a 332 MB delete. |
| [AFC delete leaves ghosts](./afc-delete-ghosts) | AFC delete frees DCIM bytes but leaves Photos.sqlite ZASSET rows. Ghosts appear with exclamation marks. Tapping clears the row; bulk clear from app-side is what users actually want. |
| [VTracer for raster→SVG](./vtracer-for-raster-to-svg) | Hand-authored SVG paths always look like ass compared to the source PNG. `cargo install vtracer` produces a 37 KB pixel-faithful SVG. Never hand-trace again. |

## Adding a new discovery

1. Pick a kebab-case filename describing the topic, not the project.
2. Lead with the conclusion.
3. Show the test or measurement that proves it.
4. Link to the relevant code path (`internal/foo/bar.go:42`).
5. Add a row to the table above.
6. Add a sidebar link in `docs/internal/.vitepress/config.ts`.
