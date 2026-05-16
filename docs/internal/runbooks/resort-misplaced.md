# Runbook — Re-sort misplaced files

When a previous backup put files in the wrong `YYYY-MM-DD/` folder — usually because of the `-fast2` exiftool trap or a corrupted MOV moov atom — this is how to fix the existing tree without re-pulling.

## When to run this

- The destination has files lumped under `0000:00:00 00:00:00/` that should have real dates.
- The destination has MOV files dated to "today" or whenever the backup ran, instead of capture date.
- You upgraded to a DumpSock build with a smarter EXIF reader and want to retroactively fix the old layout.

## What it does

Walks the destination tree, re-reads EXIF / moov capture dates from each file with our current pure-Go EXIF reader, computes the correct `YYYY-MM-DD/` folder, and `mv`s misplaced files. Live Photo pairs are kept together.

## Running

```bash
dumpsock resort \
  --root ~/DumpSock/iPhone \
  --dry-run            # always preview first
```

Then for real:

```bash
dumpsock resort --root ~/DumpSock/iPhone
```

Flags:
- `--root DIR` (required) — the backup root with `YYYY-MM-DD/` subfolders.
- `--dry-run` — preview moves, write nothing.
- `--no-mtime` — don't fix mtime on the moved files.
- `--keep-empty` — don't remove empty source folders after move.

## Expected output

```
scanning ~/DumpSock/iPhone ...
inspected 12,304 files; will move 487
  IMG_4321.HEIC: 2026-05-15 → 2026-05-12
  IMG_4321.MOV : 2026-05-15 → 2026-05-12   (paired with HEIC)
  IMG_5012.MOV : 0000-00-00 → 2025-08-30
  ...
moved 487 files in 4.1s. 12 source folders are now empty (removed).
```

## Safety properties

- Reads + plans before moving anything. `--dry-run` does the same work minus the moves.
- Refuses to overwrite — if a file with the same name already exists at the target, the existing one wins and the source gets a `-1` (`-2`, …) suffix appended.
- Operates on `os.Rename` only (same filesystem). Refuses to operate across mount points — explicit error.
- Does NOT touch files outside `--root`.
- Does NOT touch `.AAE` sidecars without their parent media file (they're moved together).

## Common pitfalls

- **"Source and destination on different volumes"** — `os.Rename` only works within a single filesystem. If you're consolidating two backups onto one drive, copy first.
- **"Operation not permitted"** — the destination is owned by a different user, or you're on a Lexar SSD with weird ACLs. `chmod -R u+w` the tree first.
- **"No moves planned"** — either the current EXIF reader is reading the same dates the previous run wrote (consistent → nothing to fix), or all files are already in `YYYY-MM-DD/` matching their EXIF.

## Provenance

The `resort` subcommand was born from the exiftool `-fast2` trap discovery — see [exiftool -fast2 trap](../discoveries/exiftool-fast2-trap). After we fixed the bug, the operator had ~500 MOV files dated to the day-of-backup. Rather than re-pull 70 GB, we wrote `resort`. The same command can recover from any future EXIF reader fix.

## What it does NOT fix

- Filenames that were collision-suffixed (`-1`, `-2`) when they shouldn't have been.
- Files that fell back to mtime correctly the first time and should stay there.
- Metadata inside the files (we never write EXIF to files; we just read).
