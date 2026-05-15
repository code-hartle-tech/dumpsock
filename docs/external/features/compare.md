# Compare & Merge

Side-by-side diff between your iPhone and the destination folder.

## What it shows

- **Source (Device):** total items + size on the iPhone.
- **Destination (Backup):** total items + size on your disk.
- Per-category split: Photos, Videos, Live Photos, Screenshots, Documents, Other.
- Three filter pills:
  - **New in Device** — on the phone but not yet at your destination.
  - **New in Backup** — at your destination but not on the phone (could be older photos you've archived).
  - **Different** — same filename but a different byte size somewhere.

## How the matching works

DumpSock matches files by **filename + byte size**, walking your destination tree once at startup and the iPhone's DCIM via AFC. The hash mode is fast and works for almost every real-world case (iPhone filenames are unique within DCIM; matching by both name and size catches edge cases where the same name was reused).

For paranoid use cases, `dumpsock pull --hash sha256` will eventually swap this for content-hashing (issue #2 — not yet shipped).

## Status

Compare & Merge UI is live; live counts will be wired up as soon as the in-flight engine work in [#2](https://github.com/code-hartle-tech/dumpsock/issues/2) lands. Until then the panel shows representative numbers from a sample run.
