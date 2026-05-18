package backup

import (
	"archive/zip"
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

	"golang.org/x/crypto/pbkdf2"

	_ "crypto/sha256" // PBKDF2-SHA256
	"crypto/sha256"
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
func PackageZip(ctx context.Context, outputRoot string, onProgress func(PackageProgress)) (string, error) {
	if outputRoot == "" {
		return "", errors.New("PackageZip: empty outputRoot")
	}
	cleaned := filepath.Clean(outputRoot)
	leaf := filepath.Base(cleaned)
	zipPath := freeName(filepath.Join(cleaned, leaf+".zip"))

	// Pre-walk: sum total bytes so the progress bar has a real
	// denominator. Cheap (one Stat per file) compared to the actual
	// copy step. Same skip rules as the main walk below.
	var bytesTotal int64
	_ = filepath.WalkDir(cleaned, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".dumpsock") ||
			strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".zip.aes") {
			return nil
		}
		if info, _ := d.Info(); info != nil {
			bytesTotal += info.Size()
		}
		return nil
	})
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
	walkErr := filepath.WalkDir(cleaned, func(p string, d fs.DirEntry, err error) error {
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
		// Skip the zip we're currently writing AND any older archive
		// files that happen to live in outputRoot (the .zip / .zip.aes
		// from a prior run) so re-running a backup doesn't gobble its
		// own predecessors into the new archive.
		if !d.IsDir() {
			cp := filepath.Clean(p)
			if cp == zipPath ||
				strings.HasSuffix(name, ".zip") ||
				strings.HasSuffix(name, ".zip.aes") {
				return nil
			}
		}
		rel, _ := filepath.Rel(cleaned, p)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			_, err := zw.Create(rel + "/")
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		// zip.Store = no compression. Already-compressed media (HEIC,
		// JPEG, H.264) gains <1% from Deflate and costs CPU-hours on a
		// real-world backup. Operator-reported "packaging takes
		// FOREVER" on 2026-05-18 — fixed by switching to Store.
		header.Method = zip.Store
		w, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		f, err := os.Open(p)
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
						Current:    name,
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
	})
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
// .aes file.
//
// onProgress (nullable) reports cumulative-plaintext-bytes-processed,
// emitted after each chunk seal.
func EncryptFile(src, password string, onProgress func(PackageProgress)) (string, error) {
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

	dst := src + ".aes"
	out, err := os.Create(dst)
	if err != nil {
		return "", fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()

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
