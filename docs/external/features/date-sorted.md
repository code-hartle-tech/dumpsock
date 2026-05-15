# Date-sorted output

DumpSock organizes every pulled file into a folder named after its **capture date** in `YYYY-MM-DD` format.

```
~/DumpSock/iPhone/
├── 2026-05-15/
│   ├── IMG_0107.HEIC
│   └── IMG_0107.MOV
├── 2026-05-14/
│   └── …
└── 0000:00:00 00:00:00/
    └── files-with-no-readable-date.ext
```

## Date precedence

For each file, DumpSock reads metadata in this order:

1. **`DateTimeOriginal`** (EXIF; photos)
2. **`CreateDate`** (EXIF / QuickTime; videos)
3. **`MediaCreateDate`** (QuickTime; older videos)
4. **`FileModifyDate`** (filesystem fallback)

The first usable value wins. If none parse, the file lands in `0000:00:00 00:00:00/` (a placeholder folder name we copied from icloudpd's behavior — same string, so the two tools interoperate).

## Live Photos stay together

A Live Photo is two files: `IMG_0042.HEIC` (the still) plus `IMG_0042.MOV` (the motion). Both share the same `DateTimeOriginal`, so they land in the same date folder side-by-side.

## File mtime

By default DumpSock sets each file's filesystem `mtime` to its capture date. This means:

- Finder's "Date Modified" column shows when the photo was taken, not when DumpSock wrote it.
- `ls -l --sort=time` orders files chronologically.
- Time Machine / rsync incremental backups can use mtime for efficient sync.

If you don't want this, run with `--no-mtime`.

## Name collisions

If a file with the same name **and same byte size** already exists in the destination tree, DumpSock skips it (it's the same file). If the name matches but the size differs, DumpSock appends `-1`, `-2`, ... so nothing gets silently overwritten:

```
IMG_0042.HEIC      # original
IMG_0042-1.HEIC    # different content, same iPhone filename (rare but possible)
```

This matches icloudpd's `name-size-dedup-with-suffix` policy exactly.
