package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	zip "github.com/yeka/zip"
)

// ExtractZip unpacks srcZip into dstDir. Handles password-encrypted
// zips (yeka/zip's AES support) when password is non-empty; calling
// with an empty password against a plain zip is also fine.
//
// dstDir must not exist — we refuse to merge into an existing tree
// to avoid silently clobbering. Callers should auto-suffix via
// nextFreeName-style logic.
//
// Returns the count of files extracted on success.
func ExtractZip(ctx context.Context, srcZip, dstDir, password string, onProgress func(PackageProgress)) (count int, err error) {
	if srcZip == "" {
		return 0, errors.New("ExtractZip: empty srcZip")
	}
	if dstDir == "" {
		return 0, errors.New("ExtractZip: empty dstDir")
	}
	if _, statErr := os.Stat(dstDir); statErr == nil {
		return 0, fmt.Errorf("destination already exists: %s", dstDir)
	}
	if err = os.MkdirAll(dstDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir dst: %w", err)
	}
	// Rollback the partially-extracted folder on any failure (cancel,
	// permission error, corrupt zip entry, etc.). Operator preference
	// 2026-05-18 — "cancel should rollback whatever it is."
	defer func() {
		if err != nil {
			_ = os.RemoveAll(dstDir)
		}
	}()

	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return 0, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	total := int64(0)
	for _, f := range r.File {
		if !f.FileInfo().IsDir() {
			total += int64(f.UncompressedSize64)
		}
	}
	if onProgress != nil {
		onProgress(PackageProgress{Phase: "extracting", BytesTotal: total})
	}

	var bytesDone int64
	var fileCount int
	for _, f := range r.File {
		if ctx.Err() != nil {
			return fileCount, ctx.Err()
		}
		// Defend against zip-slip: reject entries whose cleaned path
		// escapes dstDir.
		target := filepath.Join(dstDir, f.Name)
		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), filepath.Clean(dstDir)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(dstDir) {
			return fileCount, fmt.Errorf("refusing path outside destination: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return fileCount, fmt.Errorf("mkdir %s: %w", target, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fileCount, fmt.Errorf("mkdir parent of %s: %w", target, err)
		}
		if password != "" && f.IsEncrypted() {
			f.SetPassword(password)
		}
		rc, err := f.Open()
		if err != nil {
			return fileCount, fmt.Errorf("open zip entry %s: %w (wrong password?)", f.Name, err)
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			_ = rc.Close()
			return fileCount, fmt.Errorf("create %s: %w", target, err)
		}
		// Stream copy with periodic progress emits (every ~16 MiB).
		buf := make([]byte, 1<<20)
		for {
			n, rerr := rc.Read(buf)
			if n > 0 {
				if _, werr := out.Write(buf[:n]); werr != nil {
					_ = out.Close()
					_ = rc.Close()
					return fileCount, fmt.Errorf("write %s: %w", target, werr)
				}
				bytesDone += int64(n)
				if onProgress != nil {
					onProgress(PackageProgress{
						Phase:      "extracting",
						BytesDone:  bytesDone,
						BytesTotal: total,
						Current:    f.Name,
					})
				}
			}
			if rerr == io.EOF {
				break
			}
			if rerr != nil {
				_ = out.Close()
				_ = rc.Close()
				return fileCount, fmt.Errorf("read zip entry %s: %w", f.Name, rerr)
			}
		}
		_ = out.Close()
		_ = rc.Close()
		// Preserve the original mtime so date-sort tools still work.
		if mt := f.ModTime(); !mt.IsZero() {
			_ = os.Chtimes(target, mt, mt)
		}
		fileCount++
	}
	return fileCount, nil
}
