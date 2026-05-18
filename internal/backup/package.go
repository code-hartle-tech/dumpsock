package backup

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	// yeka/zip is a fork of stdlib archive/zip with AES-256 password
	// encryption added. We always use this — when no password is set,
	// behaviour is byte-identical to archive/zip (Store mode); when a
	// password is set, per-file AES-256 encryption is applied.
	zip "github.com/yeka/zip"
	"golang.org/x/crypto/pbkdf2"

	_ "crypto/sha256" // PBKDF2-SHA256 import for side-effect registration
	"crypto/sha256"
)

// PasswordMode is how the password (if any) is applied to the archive.
type PasswordMode string

const (
	// PasswordNone — no encryption. Bundle into a Store-mode .zip and
	// stop there. Fastest path.
	PasswordNone PasswordMode = ""

	// PasswordStandard — zip-native AES-256 per file. ANY zip tool with
	// AES support (7-Zip, Keka, modern macOS Finder, WinRAR) can decrypt
	// with the password. Portable; file list metadata (names, sizes)
	// is visible without the password, contents are not.
	PasswordStandard PasswordMode = "standard"

	// PasswordMaximum — zip-native AES-256 + DSAES2 wrapper on top.
	// Slower (two layers) but only DumpSock decrypts the result, and
	// the outer wrapper hides the file list entirely. Defense-in-depth.
	PasswordMaximum PasswordMode = "maximum"
)

// PackageProgress is fed to PackageZip's / EncryptFile's OnProgress
// callback to drive a live progress bar in the GUI. BytesDone is the
// cumulative count of plaintext bytes processed; BytesTotal is the
// sum of all files in the walk (computed by a pre-pass before the
// archive write begins, so the bar has a real denominator).
type PackageProgress struct {
	Phase     string // "packaging" or "encrypting"
	BytesDone int64
	BytesTotal int64
	Current   string // file currently being added (packaging only)
}

// PackageZip walks outputRoot and writes a .zip archive **inside** it
// that contains every other file under it (skipping the .dumpsock-*
// sentinels and any older .zip/.zip.aes archives sitting alongside).
// Returns the path of the produced .zip. Honors ctx cancellation
// between files.
//
// onProgress (nullable) is invoked periodically with the cumulative
// byte count. The GUI uses this to render a live progress bar so the
// operation doesn't feel like a hang on multi-GB backups.
//
// LOCATION: operator preference 2026-05-18 — the archive lands at
// `<outputRoot>/<leaf>.zip` (inside the backup folder) rather than as
// a sibling. Browsing outputRoot in Finder shows the archive right
// next to the YYYY-MM-DD/ subfolders, instead of forcing the user to
// step up a directory to find it.
//
// PERFORMANCE: every entry uses `zip.Store` (no recompression). DumpSock
// backups are dominated by HEIC, JPEG, H.264, and HEVC content — all
// already compressed. Running Deflate over them gains <1% size and
// burns enormous CPU for tens of GB of media. Store turns the whole
// step into disk-I/O-bound bytes-shoveling.
//
// Naming: outputRoot=/x/y/MyPhone → /x/y/MyPhone/MyPhone.zip. Conflicts
// (e.g. an old archive from a previous run) get -1, -2, … suffix.
//
// password / mode behaviour:
//   - mode==PasswordNone or password=="" → plain Store-mode zip, no
//     encryption. Fastest path.
//   - mode==PasswordStandard | PasswordMaximum → each file is encrypted
//     with per-file AES-256 using the password (WinZip/PKWare AE-2,
//     decryptable by 7-Zip / Keka / modern Finder). Standard finishes
//     here; Maximum's caller then runs EncryptFile on the produced zip
//     to add a DSAES2 wrapper.
// onlyPaths, when non-nil, restricts the archive to those exact local
// paths (skipping the recursive walk). Used by the GUI's "Only this
// run" scope toggle which feeds Result.PulledPaths from the just-
// completed pull. Each path must be absolute and live under outputRoot;
// archive entries are encoded relative to outputRoot so the resulting
// zip preserves the YYYY-MM-DD/ structure.
func PackageZip(ctx context.Context, outputRoot, password string, mode PasswordMode, onlyPaths []string, onProgress func(PackageProgress)) (string, error) {
	if outputRoot == "" {
		return "", errors.New("PackageZip: empty outputRoot")
	}
	encrypt := password != "" && mode != PasswordNone
	scoped := len(onlyPaths) > 0
	cleaned := filepath.Clean(outputRoot)
	leaf := filepath.Base(cleaned)
	zipPath := freeName(filepath.Join(cleaned, leaf+".zip"))

	// Pre-walk: sum total bytes so the progress bar has a real
	// denominator. Cheap (one Stat per file) compared to the actual
	// copy step.
	var bytesTotal int64
	if scoped {
		for _, p := range onlyPaths {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				bytesTotal += info.Size()
			}
		}
	} else {
		_ = filepath.WalkDir(cleaned, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".dumpsock") ||
				strings.HasSuffix(name, ".zip") ||
				strings.HasSuffix(name, ".zip.aes") ||
				strings.HasSuffix(name, ".dumpsock") {
				return nil
			}
			if info, _ := d.Info(); info != nil {
				bytesTotal += info.Size()
			}
			return nil
		})
	}
	if onProgress != nil {
		onProgress(PackageProgress{Phase: "packaging", BytesTotal: bytesTotal})
	}

	out, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("create zip: %w", err)
	}
	defer out.Close()

	var bytesDone int64
	lastEmit := bytesDone
	zw := zip.NewWriter(out)

	// addFile streams one file into the zip with the right header
	// settings (Store + optional AES-256). rel is the path inside the
	// archive; absPath is the on-disk source.
	addFile := func(absPath, rel, displayName string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return err
		}
		if info.IsDir() {
			_, err := zw.Create(rel + "/")
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		// Store mode keeps the operation disk-bound; already-compressed
		// media (HEIC, JPEG, H.264) gains <1% from Deflate at huge CPU
		// cost. Per-file AES (~few GB/s with AES-NI) is much cheaper.
		header.Method = zip.Store
		if encrypt {
			header.SetPassword(password)
			header.SetEncryptionMethod(zip.AES256Encryption)
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(absPath)
		if err != nil {
			return err
		}
		// Stream copy with periodic progress emits (every ~16 MiB).
		buf := make([]byte, 1<<20)
		var copyErr error
		for {
			n, rErr := f.Read(buf)
			if n > 0 {
				if _, wErr := w.Write(buf[:n]); wErr != nil {
					copyErr = wErr
					break
				}
				bytesDone += int64(n)
				if onProgress != nil && bytesDone-lastEmit >= 16<<20 {
					onProgress(PackageProgress{
						Phase:      "packaging",
						BytesDone:  bytesDone,
						BytesTotal: bytesTotal,
						Current:    displayName,
					})
					lastEmit = bytesDone
				}
			}
			if rErr == io.EOF {
				break
			}
			if rErr != nil {
				copyErr = rErr
				break
			}
		}
		_ = f.Close()
		return copyErr
	}

	var walkErr error
	if scoped {
		// Scoped mode: only the explicit paths get archived. Used by
		// the "Only this run" toggle in the GUI; res.PulledPaths from
		// the just-completed pull is passed through.
		for _, abs := range onlyPaths {
			cp := filepath.Clean(abs)
			rel, relErr := filepath.Rel(cleaned, cp)
			if relErr != nil || strings.HasPrefix(rel, "..") {
				// Path is outside outputRoot — skip rather than escape.
				continue
			}
			if err := addFile(cp, rel, filepath.Base(cp)); err != nil {
				walkErr = err
				break
			}
		}
	} else {
		walkErr = filepath.WalkDir(cleaned, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			name := d.Name()
			// Skip DumpSock-internal sentinels / staging dirs.
			if strings.HasPrefix(name, ".dumpsock") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// Skip the zip we're currently writing AND any older
			// archive files in outputRoot (so re-running a backup
			// doesn't gobble its own predecessors into the new zip).
			if !d.IsDir() {
				cp := filepath.Clean(p)
				if cp == zipPath ||
					strings.HasSuffix(name, ".zip") ||
					strings.HasSuffix(name, ".zip.aes") ||
					strings.HasSuffix(name, ".dumpsock") {
					return nil
				}
			}
			rel, _ := filepath.Rel(cleaned, p)
			if rel == "." {
				return nil
			}
			return addFile(p, rel, name)
		})
	}
	if onProgress != nil {
		onProgress(PackageProgress{
			Phase:      "packaging",
			BytesDone:  bytesDone,
			BytesTotal: bytesTotal,
		})
	}
	closeErr := zw.Close()
	if walkErr != nil {
		return "", walkErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return zipPath, nil
}

// Encryption container — DSAES2 (chunked, streaming).
//
// PERFORMANCE: the original DSAES1 format encrypted in one shot, which
// required reading the entire archive into RAM (os.ReadFile) and one
// monster AES-GCM Seal call. That's fine for a 100 MB archive; brutal
// for a 50 GB photo library. DSAES2 streams the file in fixed-size
// chunks, each chunk independently sealed with its own nonce + tag.
//
// On-disk layout:
//
//	magic       "DSAES2\n"     7 bytes
//	salt        16 bytes       random per file
//	iters       uint32 BE      PBKDF2-SHA256 iterations (currently 200,000)
//	chunkSize   uint32 BE      bytes of plaintext per chunk (we use 1 MiB)
//	[ chunks... ]              each chunk = nonce(12) || ct(<=chunkSize+16-tag)
//
// The final chunk may have a shorter plaintext (no padding). Reader
// concatenates the decrypted plaintexts to reconstruct the archive.
const (
	dsaes2Magic     = "DSAES2\n"
	dsaes2Iters     = 200_000
	dsaes2ChunkSize = 1 << 20 // 1 MiB
)

// EncryptFile streams src through PBKDF2-derived AES-256-GCM into a
// sibling file with ".aes" appended. Returns the path of the produced
// .aes file. Honors ctx: a cancellation between chunks aborts the
// write AND removes the partial .aes from disk.
//
// onProgress (nullable) reports cumulative-plaintext-bytes-processed,
// emitted after each chunk seal.
func EncryptFile(ctx context.Context, src, password string, onProgress func(PackageProgress)) (dst string, err error) {
	if password == "" {
		return "", errors.New("EncryptFile: empty password")
	}
	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	stat, _ := in.Stat()
	bytesTotal := int64(0)
	if stat != nil {
		bytesTotal = stat.Size()
	}
	if onProgress != nil {
		onProgress(PackageProgress{Phase: "encrypting", BytesTotal: bytesTotal})
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := pbkdf2.Key([]byte(password), salt, dsaes2Iters, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	// Branded extension (operator preference 2026-05-18): replace
	// `.zip` with `.dumpsock`; if src has no `.zip` suffix just append.
	// Existing `.zip.aes` archives keep decrypting via the magic-byte
	// header check in DecryptFile — only NEW archives get the new ext.
	dst = strings.TrimSuffix(src, ".zip") + ".dumpsock"
	if dst == src+".dumpsock" {
		// src had no .zip suffix; the produced file becomes "src.dumpsock"
	}
	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()
	// Roll back the partial output on any failure (cancel, IO error,
	// short read, etc.) — operator preference 2026-05-18 ("cancel
	// should rollback/remove whatever it is").
	defer func() {
		if err != nil {
			_ = os.Remove(dst)
		}
	}()

	// Header.
	if _, err := out.Write([]byte(dsaes2Magic)); err != nil {
		return "", err
	}
	if _, err := out.Write(salt); err != nil {
		return "", err
	}
	hdrBuf := make([]byte, 8)
	binary.BigEndian.PutUint32(hdrBuf[0:4], uint32(dsaes2Iters))
	binary.BigEndian.PutUint32(hdrBuf[4:8], uint32(dsaes2ChunkSize))
	if _, err := out.Write(hdrBuf); err != nil {
		return "", err
	}

	// Stream chunks.
	plain := make([]byte, dsaes2ChunkSize)
	nonce := make([]byte, aead.NonceSize())
	var bytesDone int64
	for {
		if ctx != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return "", ctxErr
			}
		}
		n, readErr := io.ReadFull(in, plain)
		// io.ReadFull returns ErrUnexpectedEOF when it gets a partial
		// final read — that's the last chunk and is fine.
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return "", fmt.Errorf("read src: %w", readErr)
		}
		if n == 0 {
			break
		}
		if _, err := rand.Read(nonce); err != nil {
			return "", fmt.Errorf("nonce: %w", err)
		}
		ct := aead.Seal(nil, nonce, plain[:n], nil)
		if _, err := out.Write(nonce); err != nil {
			return "", err
		}
		if _, err := out.Write(ct); err != nil {
			return "", err
		}
		bytesDone += int64(n)
		if onProgress != nil {
			onProgress(PackageProgress{
				Phase:      "encrypting",
				BytesDone:  bytesDone,
				BytesTotal: bytesTotal,
			})
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	return dst, nil
}

// freeName returns base if base doesn't exist, otherwise base with a
// -1, -2, ... suffix inserted before the extension to find an unused
// filename. Bounded at 1000 attempts.
func freeName(base string) string {
	if _, err := os.Stat(base); err != nil {
		return base
	}
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 1; i < 1000; i++ {
		cand := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, err := os.Stat(cand); err != nil {
			return cand
		}
	}
	return fmt.Sprintf("%s-%d%s", stem, 999, ext)
}
