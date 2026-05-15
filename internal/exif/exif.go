// Package exif extracts capture dates from media files.
//
// Pure-Go, no external binary, no install step:
//
//   - HEIC / HEIF / JPEG / PNG / TIFF / DNG / CR2 / CR3 / NEF / ARW:
//     github.com/evanoberholster/imagemeta. We read DateTimeOriginal,
//     fall back to CreateDate, then ModifyDate.
//   - MOV / MP4 / M4V: minimal ISO-BMFF atom walker built in (see
//     parseQuickTime); reads moov/mvhd creation_time. QuickTime time
//     epoch is 1904-01-01 UTC, hence the offset constant below.
//   - Anything else: file mtime (os.Stat) as last-resort fallback.
//
// Returns (zeroTime, false, nil) only when ALL sources fail or the file
// truly has no readable date metadata.
package exif

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanoberholster/imagemeta"
)

// QuickTime / ISO-BMFF epoch: 1904-01-01 UTC. Seconds offset to Unix epoch.
const quickTimeEpochOffset int64 = 2082844800

// Reader extracts capture dates. Zero-state, no construction needed but
// kept as a struct to preserve the v0 API shape and leave room for
// future configuration (timezone overrides, per-format toggles, etc.).
type Reader struct{}

// NewReader is kept for backward compatibility with the exiftool-shelled
// implementation. There is no longer any external dependency to validate;
// the returned reader is always usable.
func NewReader() (*Reader, error) {
	return &Reader{}, nil
}

// SetBin is a no-op retained for API compatibility with the previous
// exiftool-based Reader. Pure-Go implementation has no binary to point at.
func (r *Reader) SetBin(_ string) {}

// Date reads the most-authoritative capture date from path.
//
// Precedence (most authoritative first):
//
//	images: DateTimeOriginal > CreateDate > ModifyDate
//	video:  moov/mvhd creation_time
//	any:    file mtime as last-resort fallback
//
// Returns (zeroTime, false, nil) only when even the fallback fails.
func (r *Reader) Date(ctx context.Context, path string) (time.Time, bool, error) {
	if err := ctx.Err(); err != nil {
		return time.Time{}, false, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mov", ".mp4", ".m4v":
		if t, ok := readQuickTime(path); ok {
			return t, true, nil
		}
	default:
		if t, ok := readImageMeta(path); ok {
			return t, true, nil
		}
	}

	// Fallback to file mtime — guarantees a usable date for almost any file.
	if info, err := os.Stat(path); err == nil {
		return info.ModTime(), true, nil
	}
	return time.Time{}, false, nil
}

// readImageMeta uses imagemeta.Decode on the file. Returns (zero, false)
// if the file isn't a supported image type or has no readable EXIF.
func readImageMeta(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()

	e, err := imagemeta.Decode(f)
	if err != nil {
		return time.Time{}, false
	}
	// Most-authoritative tag first.
	if !e.ExifIFD.DateTimeOriginal.IsZero() {
		return e.ExifIFD.DateTimeOriginal, true
	}
	if !e.ExifIFD.CreateDate.IsZero() {
		return e.ExifIFD.CreateDate, true
	}
	if !e.IFD0.ModifyDate.IsZero() {
		return e.IFD0.ModifyDate, true
	}
	return time.Time{}, false
}

// readQuickTime walks the MP4 / MOV box tree looking for moov/mvhd and
// returns the creation_time as a UTC time.Time.
//
// Reference: ISO/IEC 14496-12, §8.2 (mvhd box). Box header is
//
//	uint32 size  // includes header
//	char4  type
//
// size==1 means "extended": next 8 bytes are a uint64 with the true size.
// size==0 means "this box extends to EOF".
//
// mvhd payload:
//
//	uint8  version (0 or 1)
//	uint24 flags
//	(v0) uint32 creation_time, modification_time, timescale, duration
//	(v1) uint64 creation_time, modification_time, uint32 timescale, uint64 duration
func readQuickTime(path string) (time.Time, bool) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return time.Time{}, false
	}
	ct, ok := walkForMvhd(f, info.Size())
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(int64(ct)-quickTimeEpochOffset, 0).UTC(), true
}

// walkForMvhd descends into top-level boxes until it finds moov, then
// recurses inside moov to find mvhd and returns its creation_time.
func walkForMvhd(rs io.ReadSeeker, end int64) (uint64, bool) {
	return walkBoxes(rs, 0, end, func(typ string, payloadStart, payloadEnd int64) (uint64, bool, bool) {
		switch typ {
		case "moov":
			// Recurse into moov.
			ct, ok := walkBoxes(rs, payloadStart, payloadEnd, func(typ2 string, ps2, pe2 int64) (uint64, bool, bool) {
				if typ2 == "mvhd" {
					ct, ok := readMvhd(rs, ps2, pe2)
					return ct, ok, true // stop on first mvhd
				}
				return 0, false, false
			})
			return ct, ok, true
		}
		return 0, false, false
	})
}

// walkBoxes iterates top-level boxes inside [start, end). The handler
// is called for each box with (type, payloadStart, payloadEnd). It
// returns (foundValue, found, stop). If stop is true, iteration ends
// immediately with the returned value.
func walkBoxes(rs io.ReadSeeker, start, end int64, fn func(typ string, payloadStart, payloadEnd int64) (uint64, bool, bool)) (uint64, bool) {
	if _, err := rs.Seek(start, io.SeekStart); err != nil {
		return 0, false
	}
	pos := start
	for pos < end {
		if _, err := rs.Seek(pos, io.SeekStart); err != nil {
			return 0, false
		}
		var hdr [8]byte
		if _, err := io.ReadFull(rs, hdr[:]); err != nil {
			return 0, false
		}
		size := uint64(binary.BigEndian.Uint32(hdr[:4]))
		typ := string(hdr[4:8])
		payloadStart := pos + 8

		if size == 1 {
			var ext [8]byte
			if _, err := io.ReadFull(rs, ext[:]); err != nil {
				return 0, false
			}
			size = binary.BigEndian.Uint64(ext[:])
			payloadStart = pos + 16
		} else if size == 0 {
			size = uint64(end - pos)
		}
		if size < 8 {
			return 0, false // malformed
		}
		boxEnd := pos + int64(size)
		if boxEnd > end {
			return 0, false
		}
		if val, ok, stop := fn(typ, payloadStart, boxEnd); stop {
			return val, ok
		}
		pos = boxEnd
	}
	return 0, false
}

// readMvhd parses the mvhd payload at [start, end) and returns its
// creation_time.
func readMvhd(rs io.ReadSeeker, start, end int64) (uint64, bool) {
	if _, err := rs.Seek(start, io.SeekStart); err != nil {
		return 0, false
	}
	var hdr [4]byte
	if _, err := io.ReadFull(rs, hdr[:]); err != nil {
		return 0, false
	}
	version := hdr[0]
	switch version {
	case 0:
		var t32 [4]byte
		if _, err := io.ReadFull(rs, t32[:]); err != nil {
			return 0, false
		}
		return uint64(binary.BigEndian.Uint32(t32[:])), true
	case 1:
		var t64 [8]byte
		if _, err := io.ReadFull(rs, t64[:]); err != nil {
			return 0, false
		}
		return binary.BigEndian.Uint64(t64[:]), true
	default:
		return 0, false
	}
}

// ErrExiftoolNotFound is retained for API compatibility but is never
// returned by the pure-Go implementation. New code shouldn't reference it.
var ErrExiftoolNotFound = errors.New("exiftool not installed (legacy error — pure-Go reader is used now)")
