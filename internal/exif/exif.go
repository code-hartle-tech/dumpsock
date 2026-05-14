// Package exif extracts capture dates from media files.
//
// v0 shells out to `exiftool` for HEIC/JPEG/PNG/MOV/MP4/DNG coverage. The
// Phase 1 plan calls for swapping to a pure-Go reader (dsoprea/go-exif +
// QuickTime atom parsing) so the final DumpSock binary is fully
// self-contained.
package exif

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// ErrExiftoolNotFound is returned when the `exiftool` binary isn't on PATH.
var ErrExiftoolNotFound = errors.New("exiftool not installed (brew install exiftool)")

// Reader reads capture dates from media files.
type Reader struct {
	bin string
}

// NewReader resolves the `exiftool` binary on $PATH. Caller can override the
// resolved path via SetBin if a non-default location is needed.
func NewReader() (*Reader, error) {
	bin, err := exec.LookPath("exiftool")
	if err != nil {
		return nil, ErrExiftoolNotFound
	}
	return &Reader{bin: bin}, nil
}

// SetBin overrides the exiftool binary path.
func (r *Reader) SetBin(path string) { r.bin = path }

// Date reads the most-authoritative capture date from path. Precedence
// matches icloudpd: DateTimeOriginal > CreateDate > MediaCreateDate >
// FileModifyDate. Returns (zero time, false, nil) when no date can be read.
//
// The context is honored — long-running exiftool calls can be cancelled.
func (r *Reader) Date(ctx context.Context, path string) (time.Time, bool, error) {
	cmd := exec.CommandContext(ctx, r.bin,
		"-s3", "-fast2",
		"-d", "%Y-%m-%dT%H:%M:%S",
		"-FileModifyDate",
		"-MediaCreateDate",
		"-CreateDate",
		"-DateTimeOriginal",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		// exiftool returns non-zero for unreadable files; treat as "no date".
		return time.Time{}, false, nil
	}

	// Output is one line per requested tag, in the order requested. Later
	// (more authoritative) values shadow earlier ones if both present.
	var best time.Time
	var found bool
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "-" || strings.HasPrefix(line, "0000") {
			continue
		}
		t, ok := parseExifTime(line)
		if !ok {
			continue
		}
		best = t
		found = true
	}
	return best, found, nil
}

func parseExifTime(s string) (time.Time, bool) {
	// Tolerated forms:
	//   2026:05:14 19:01:00
	//   2026-05-14T19:01:00
	//   2026-05-14
	formats := []string{
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006:01:02 15:04:05",
		"2006-01-02",
		"2006:01:02",
	}
	for _, f := range formats {
		if t, err := time.ParseInLocation(f, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
