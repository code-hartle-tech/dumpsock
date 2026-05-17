# Compressed + password-protected archives

DumpSock's default is a plain folder tree on disk — exactly what most people want. But if you're moving a backup to a USB drive, emailing it to yourself, or stashing it on cloud storage you don't fully trust, the **Compress** and **Password-protect** options on the Dashboard package the whole backup into a single encrypted file.

## When to use it

- **Moving to portable media.** A single `.zip` is easier to copy than a folder of 5,000 files.
- **Sending across an untrusted boundary.** Encrypted, you can put it on Dropbox / iCloud Drive / a thumb drive at the airport.
- **Long-term cold storage.** Tape, optical disc, Glacier — one self-contained artifact you can verify integrity on.

## How it works

After a normal backup completes, DumpSock walks the output folder and produces a sibling archive:

1. **Compress** alone → `<output>.zip` next to your backup folder.
2. **Compress + Password-protect** → `<output>.zip.aes`. The plain zip is created first, then encrypted with AES-256-GCM and the intermediate zip is removed. You're left with a single encrypted file.

The done banner shows the artifact path. A toast also fires with the full path so you can find it after switching tabs.

## Encryption details

For users who want to know exactly what's happening to their bytes:

- **Cipher:** AES-256 in GCM mode (authenticated encryption — both confidentiality AND tamper detection).
- **Key derivation:** PBKDF2 with HMAC-SHA256, 200,000 iterations, 16-byte random salt, 32-byte output key.
- **Nonce:** 12 random bytes per archive (never reused).
- **Container format** (binary):

  ```
  magic   "DSAES1\n"     7 bytes
  salt    16 bytes       (random)
  iters   uint32 BE      (currently 200,000)
  nonce   12 bytes       (random, AES-GCM)
  ct‖tag  bytes          (AES-GCM ciphertext; 16-byte auth tag at the tail)
  ```

This is a deliberately small, auditable format. If you lose DumpSock you can decrypt with `openssl` + a 30-line helper script — the format is documented above, no proprietary header bytes.

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
