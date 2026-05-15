package exif

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestRealFiles exercises the pure-Go reader against representative
// iPhone-pulled samples on the operator's Lexar drive. This is a smoke
// test, not a unit test — skipped automatically when those files are not
// present, so CI doesn't break.
//
// Build a binary against your live tree:
//
//	go test -run TestRealFiles -v ./internal/exif
func TestRealFiles(t *testing.T) {
	samples := []struct {
		path string
		want string // YYYY-MM-DD expected
	}{
		{"/Volumes/Lexar/Backup/iCloud/Photos/2026-05-12/IMG_0045.MOV", "2026-05-12"},
		{"/Volumes/Lexar/Backup/iCloud/Photos/2026-05-12/IMG_0040.MOV", "2026-05-12"},
	}

	r, err := NewReader()
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}

	for _, s := range samples {
		t.Run(s.path, func(t *testing.T) {
			if _, err := os.Stat(s.path); err != nil {
				t.Skipf("sample missing: %s", s.path)
			}
			got, ok, err := r.Date(context.Background(), s.path)
			if err != nil {
				t.Fatalf("Date: %v", err)
			}
			if !ok {
				t.Fatal("no date returned for sample with known metadata")
			}
			gotStr := got.Format("2006-01-02")
			if !strings.HasPrefix(gotStr, s.want) {
				t.Fatalf("got %q, want prefix %q (full: %v)", gotStr, s.want, got)
			}
		})
	}

	// Generic stat-fallback test: any file that exists on disk should
	// yield a non-zero date via mtime.
	tmp, err := os.CreateTemp("", "dumpsock-exif-fallback-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("not a media file")
	tmp.Close()
	got, ok, err := r.Date(context.Background(), tmp.Name())
	if err != nil {
		t.Fatalf("fallback Date: %v", err)
	}
	if !ok {
		t.Fatal("fallback should always return ok=true for an existing file")
	}
	if got.IsZero() || got.After(time.Now().Add(time.Second)) {
		t.Fatalf("fallback returned suspicious time: %v", got)
	}
}
