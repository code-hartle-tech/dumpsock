package backup

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// MetadataFileName is the persistent companion file that lives in every
// completed backup directory. Unlike sessionFileName (which is a tomb-
// stone for in-progress runs), this one survives across many pulls and
// accumulates a history of when the backup was last updated.
//
// Hidden dotfile so it doesn't clutter the YYYY-MM-DD/ tree the user
// actually cares about.
const MetadataFileName = ".dumpsock.json"

// Metadata is the on-disk schema. Forward-compatible (new fields fine,
// rename/remove requires Version bump + migration). Read by the GUI
// to render the Backups list and to detect "this folder is a DumpSock
// backup of device X" when the user points at an unknown directory.
type Metadata struct {
	Version     int       `json:"version"`
	UDID        string    `json:"udid,omitempty"`
	DeviceName  string    `json:"device_name,omitempty"`
	OutputRoot  string    `json:"output_root"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TotalFiles  int       `json:"total_files"`
	TotalBytes  int64     `json:"total_bytes"`
	// History records each successful Run() against this directory.
	// Capped at 50 entries (oldest dropped) to keep the file small.
	History []RunRecord `json:"history,omitempty"`
}

// RunRecord is one entry in Metadata.History.
type RunRecord struct {
	At          time.Time     `json:"at"`
	Pulled      int           `json:"pulled"`
	PreSkipped  int           `json:"pre_skipped"`
	PostSkipped int           `json:"post_skipped"`
	Errors      int           `json:"errors"`
	BytesPulled int64         `json:"bytes_pulled"`
	Elapsed     time.Duration `json:"elapsed_ns"`
}

// ReadMetadata returns the .dumpsock.json from a directory, or
// (nil, nil) if the file isn't there. Returns a non-nil error only
// when the file exists but can't be read/parsed.
func ReadMetadata(dir string) (*Metadata, error) {
	data, err := os.ReadFile(filepath.Join(dir, MetadataFileName))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// WriteMetadata atomically writes (or overwrites) the file. Caller
// owns the merge logic — if you want to preserve history across
// updates, ReadMetadata first then pass an updated value here.
func WriteMetadata(dir string, m Metadata) error {
	if dir == "" {
		return errors.New("WriteMetadata: empty dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if m.Version == 0 {
		m.Version = 1
	}
	m.UpdatedAt = time.Now()
	if len(m.History) > 50 {
		m.History = m.History[len(m.History)-50:]
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	final := filepath.Join(dir, MetadataFileName)
	tmp, err := os.CreateTemp(dir, MetadataFileName+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, final)
}

// RecordRun loads the existing metadata at dir (or creates a fresh
// one), appends a new RunRecord describing the just-finished pull,
// and writes it back. Idempotent against missing files.
func RecordRun(dir, udid, deviceName string, r Result, bytesPulled, totalBytes int64) error {
	m, err := ReadMetadata(dir)
	if err != nil {
		// Don't fail the run for a corrupt metadata file — start fresh.
		m = nil
	}
	if m == nil {
		m = &Metadata{
			OutputRoot: dir,
			CreatedAt:  time.Now(),
		}
	}
	if udid != "" {
		m.UDID = udid
	}
	if deviceName != "" {
		m.DeviceName = deviceName
	}
	m.TotalFiles = countFilesUnder(dir)
	m.TotalBytes = totalBytes
	m.History = append(m.History, RunRecord{
		At:          time.Now(),
		Pulled:      r.Pulled,
		PreSkipped:  r.PreSkipped,
		PostSkipped: r.PostSkipped,
		Errors:      r.Errors,
		BytesPulled: bytesPulled,
		Elapsed:     r.Elapsed,
	})
	return WriteMetadata(dir, *m)
}

// countFilesUnder is a best-effort count of media files in the YYYY-MM-DD
// subtree. Skips dotfiles (so .dumpsock.json itself doesn't count) and
// dotted staging dirs. Returns 0 on traversal error rather than failing.
func countFilesUnder(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // best-effort: skip unreadable subtrees
		}
		name := d.Name()
		if p != dir && len(name) > 0 && name[0] == '.' {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n
}
