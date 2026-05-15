# Delete after backup

> **Short answer:** DumpSock can remove each photo from the iPhone **after** it's safely on your disk. Storage is freed immediately. Photos.app may show stale entries for a few minutes — that's an iOS quirk we can't bypass over USB.

See the longer write-up at [Free up iPhone storage](../guide/free-storage) for the full story (it covers the "ghost photos" issue too).

## Quick recipe

GUI: **Settings → "Wring it dry"** → run a backup → confirm the modal.

CLI: `dumpsock pull --delete-after --confirm-delete`.

## What gets deleted

Only files that DumpSock **successfully wrote to your destination** and **whose byte-size verifies against the remote** get queued for device-side deletion. Files that errored, mid-transfer drops, mismatched sizes — all skipped.

## What doesn't get deleted

- Files filtered out by `--since` / `--until`.
- Files that already existed at the destination (pre-skipped via the name+size dedup index).

Wait, the second one is a lie — those *do* get queued for deletion. The reasoning: if the file is already on your disk, the iPhone's copy is redundant.

If you don't want that, drop `--delete-after` and run a separate pass once you've verified the destination.

## What about `.AAE` sidecars?

Apple writes a small `.AAE` file alongside any photo you've cropped, rotated, or filtered. It holds the edit history. When DumpSock deletes a media file with `--delete-after`, it also sweeps the matching `.AAE` sidecar so the device doesn't end up with orphan metadata.

## Cancellation

`Ctrl-C` mid-deletion stops the deletion loop cleanly. Files that were already deleted stay deleted. Files queued but not yet deleted stay on the iPhone. Re-running the command re-queues them.

## Verification before deletion

DumpSock performs a `stat` on the just-written local file and compares its byte size to the remote file's size **before** queueing the remote for deletion. A truncated copy never triggers a delete. If you want even higher assurance, run with `--hash sha256` (slower, hashes every file).
