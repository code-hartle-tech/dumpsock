# Compressed + password-protected archives

DumpSock's default is a plain folder tree on disk — exactly what most people want. But if you're moving a backup to a USB drive, emailing it to yourself, or stashing it on cloud storage you don't fully trust, the **Compress** and **Password-protect** options on the Dashboard package the whole backup into a single encrypted file.

## When to use it

- **Moving to portable media.** A single `.zip` is easier to copy than a folder of 5,000 files.
- **Sending across an untrusted boundary.** Encrypted, you can put it on Dropbox / iCloud Drive / a thumb drive at the airport.
- **Long-term cold storage.** Tape, optical disc, Glacier — one self-contained artifact you can verify integrity on.

## How it works

After a normal backup completes, DumpSock walks the output folder and produces an archive **inside it** (right alongside the YYYY-MM-DD/ subfolders, so Finder shows the artifact in the same place you backed up to):

1. **Bundle into .zip** alone → `<output>/<name>.zip`. The .zip uses `Store` mode (no recompression) — your HEIC, JPEG, and H.264 files are already compressed, and Deflate-ing them gains <1% while burning tens of CPU-minutes on a real-world camera roll.
2. **Bundle into .zip + Password-protect** → `<output>/<name>.zip.aes`. The plain zip is produced first, then streamed through AES-256-GCM in 1 MiB chunks (so a 50 GB archive doesn't try to live in RAM), and the intermediate zip is removed. You're left with a single encrypted file.

The done banner shows the artifact path with a **Reveal archive** button that opens Finder with the file selected. A toast also fires with the full path so you can find it after switching tabs.

## Encryption details

For users who want to know exactly what's happening to their bytes:

- **Cipher:** AES-256 in GCM mode (authenticated encryption — both confidentiality AND tamper detection).
- **Key derivation:** PBKDF2 with HMAC-SHA256, 200,000 iterations, 16-byte random salt, 32-byte output key.
- **Streaming:** the archive is processed in 1 MiB chunks; each chunk is sealed with its own random 12-byte nonce + 16-byte auth tag. This is what lets DumpSock encrypt a 50 GB archive without trying to hold the whole thing in RAM.
- **Container format DSAES2** (binary):

  ```
  magic       "DSAES2\n"     7 bytes
  salt        16 bytes       (random)
  iters       uint32 BE      (currently 200,000)
  chunkSize   uint32 BE      (1 MiB = 1048576)
  [chunks…]                  each chunk = nonce(12) || ct‖tag(<= chunkSize+16)
  ```

  The reader iterates chunks until EOF. The final chunk may be shorter than the declared chunk size — no padding.

This is a deliberately small, auditable format. If you lose DumpSock you can still decrypt — the format is documented above, no proprietary header bytes.

> **Format version history:** Older builds (≤ 2026-05-18 morning) wrote `DSAES1`, a single-shot variant that fit fine for ~100 MB archives but couldn't handle GB-scale backups. Both formats are documented in-repo if you have legacy `.zip.aes` files to decrypt.

## Picking a good password

The encryption is only as strong as your password. DumpSock's password modal requires **at least 8 characters** and asks you to confirm it (typo protection). Behind the scenes, PBKDF2's 200,000 iterations make brute-forcing a strong passphrase economically prohibitive even with a GPU farm.

> **There is no recovery.** If you lose the password, the backup is unreadable — that's the entire point of encryption. Write it down somewhere physical (paper, password manager) before you click *Use this password.*

## What it does NOT protect against

- **Disk forensics on your computer** between the backup folder being written and the zip being produced. The intermediate plain zip exists briefly on disk.
- **A keylogger on your machine.** The encryption key only protects the archive, not your computer.
- **Targeted state-level adversaries.** AES-256-GCM with PBKDF2 is solid for everyday threat models; it is not a substitute for full-disk encryption + air-gapped key handling if your threat model warrants that.

## Decrypting

For now, DumpSock writes the encrypted archive but doesn't yet ship a built-in decrypt command. The format above is stable; a one-shot helper is on the roadmap. In the meantime, paste this Python snippet into a script if you need to verify:

```python
import os, sys, struct
from Crypto.Cipher import AES
from Crypto.Protocol.KDF import PBKDF2
from Crypto.Hash import SHA256
data = open(sys.argv[1], "rb").read()
assert data[:7] == b"DSAES1\n"
salt = data[7:23]
iters = struct.unpack(">I", data[23:27])[0]
nonce = data[27:39]
ct = data[39:]
key = PBKDF2(sys.argv[2].encode(), salt, dkLen=32, count=iters, hmac_hash_module=SHA256)
plain = AES.new(key, AES.MODE_GCM, nonce=nonce).decrypt_and_verify(ct[:-16], ct[-16:])
open(sys.argv[1].replace(".zip.aes", ".zip"), "wb").write(plain)
```

A polished decrypt CLI ships in the next release.
