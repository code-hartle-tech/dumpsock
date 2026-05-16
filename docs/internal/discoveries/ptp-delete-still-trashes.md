# PTP delete still trashes

## TL;DR

USB Picture Transfer Protocol (PTP) is a separate USB interface from AFC, with its own object/file model and `DeleteObject` opcode (`0x100B`). We hypothesized that PTP delete might skip Photos.app and Recently Deleted because PTP is supposed to be a generic camera protocol. **On iOS 26.4.2 it routes through PhotoKit anyway**, sending the photo to Recently Deleted with a 30-day countdown and **not freeing free space**. PTP is strictly worse than AFC delete for our use case.

## The hypothesis

PTP is designed for cameras. Cameras have storage, not "Photos apps". On a Canon DSLR, PTP `DeleteObject` simply removes the file. We thought:

> Maybe Apple wired PTP delete to the same low-level FS layer as AFC. Same delete, no PhotoKit, no Recently Deleted, immediate free space.

## The test

1. Pulled 332 MB of photos to a temp dir.
2. iPhone Storage → Photos = 71.4 GB.
3. iPhone Storage → free = 6.2 GB.
4. Issued `DeleteObject` over PTP for each photo via `libgphoto2-compatible` go-ios helper.
5. Photos.app → photos disappear from camera roll.
6. Photos.app → **Recently Deleted has 332 MB of new entries** with a 30-day timer.
7. iPhone Storage → Photos = 71.4 GB **(unchanged)**.
8. iPhone Storage → free = 6.2 GB **(unchanged)**.

After 30 days, presumably, the bytes would be freed. We waited a few minutes, then forcibly emptied Recently Deleted from inside Photos.app. THEN the bytes freed.

## Why this is worse than AFC

- **AFC delete**: bytes free immediately. Photos.app shows ghosts. Tap a ghost to clear the orphan row.
- **PTP delete**: bytes free in 30 days, OR after the user manually empties Recently Deleted (which requires Face ID, which we can't trigger over USB, which is the entire reason we're investigating this).

PTP "goes through the front door" of PhotoKit. AFC "goes around the back". For our goal — free bytes now — the back door is what we want.

## What this means

We did NOT ship PTP delete. We did NOT use `0x100B`. AFC `Remove` is the only deletion path in DumpSock.

The codebase doesn't even link the PTP wire format — go-ios's AFC service is all we use. If someone in the future thinks "let's add a PTP fallback", point them here.

## The full disagreement

There's a contradictory anecdote in the wild — some macOS Image Capture users report that "Delete after import" deletes immediately without leaving Recently Deleted entries. We tested Image Capture against the same iPhone:

- Image Capture → Import + Delete originals → Photos.app camera roll empty → Recently Deleted **has the photos** → bytes **not freed**.

So Image Capture also uses PTP, also goes through Recently Deleted, also doesn't free bytes. The anecdote was wrong, or referring to a different iOS version.

## Caveats

- We tested on iOS 26.4.2 only. Earlier iOS (especially pre-Photos.app-overhaul iOS 13) may behave differently.
- We did NOT exhaust every PTP opcode. There's a `MoveObject` (`0x1019`) and `FormatStore` (`0x100F`); we did not try them because:
  - `MoveObject` is for moving objects between PTP storage IDs, not deleting.
  - `FormatStore` would format the entire camera roll — destructive.

## Code

There is no PTP code in DumpSock. If you ever need to test this again:

```go
// scratch_only_do_not_ship.go
import "github.com/danielpaulus/go-ios/ios/ptp"
// ... PTP open + DeleteObject loop
```

Don't commit it.
