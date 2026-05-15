# CLI usage

```
dumpsock              top-level command (alias for `dumpsock pull` if one device)
dumpsock pull         the main verb
dumpsock devices      list connected iOS devices
dumpsock version      print version + brand line
dumpsock --help
```

## `dumpsock pull`

Walks the iPhone's `DCIM/` over USB, writes each file into a date-sorted folder on disk, skips what's already there.

### Day-1 flags (visible in `--help`)

| Flag | What |
|---|---|
| `-o`, `--output DIR` | Destination root. Default: `~/DumpSock/<device-name>` |
| `--since YYYY-MM-DD` | Only pull files captured on/after |
| `--until YYYY-MM-DD` | Only pull files captured on/before |
| `--delete-after` | Remove each file from device after verified copy (requires `--confirm-delete`) |
| `--confirm-delete` | Explicit confirmation gate for `--delete-after` |
| `--watch N` | Stay running, rescan every N seconds |
| `--dry-run` | Plan only — list what would be pulled, transfer nothing |
| `--udid UDID` | Disambiguate when multiple iPhones are plugged in |

### Advanced flags (hidden from default `--help`, see `--help-advanced`)

| Flag | What |
|---|---|
| `--parallel N` | Concurrent post-pull workers (EXIF read + move). Default 4. |
| `--until-found N` | Stop after N consecutive name+size matches against existing destination |
| `--no-mtime` | Don't set file mtime to capture date |
| `--no-live-pair` | Don't co-locate HEIC + MOV Live Photo pairs |
| `--no-notify` | No desktop notification on completion |
| `--hash size\|sha256` | Dedup mode. Default: name + byte size |
| `--remote-root PATH` | Path on the device to walk. Default: `DCIM` |
| `--json` | Machine-readable progress stream on stdout |

## Examples

```bash
# Pull to a specific external SSD
dumpsock pull -o /Volumes/Archive/iPhone

# Mirror only last year
dumpsock pull --since 2025-01-01 --until 2025-12-31

# Run continuously, polling every 5 minutes (cable plugged in all day)
dumpsock pull --watch 300

# Test before committing — see what would be pulled
dumpsock pull --dry-run -o /Volumes/Archive/iPhone

# Pull + free iPhone storage
dumpsock pull --delete-after --confirm-delete

# Smart resume — stop after 100 already-have-it matches (faster top-ups)
dumpsock pull --until-found 100

# Specific device
dumpsock pull --udid 00008140-001804A62209801C
```

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success (even with skipped duplicates or filter-excluded files) |
| 1 | Engine error (no device, bad flags, AFC failure, dest filesystem error) |
| 2 | User cancellation (Ctrl-C) |
