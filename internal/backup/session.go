package backup

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// sessionFileName is the sentinel file written into a backup's output
// directory while a pull is running. Its presence at app/CLI startup
// means the previous run was interrupted (crash, reboot, cable yanked,
// Ctrl-C without grace). The file is removed on clean completion.
//
// Why a hidden dotfile: keeps it out of the user's normal file browser
// view inside the YYYY-MM-DD/ tree, and the same naming convention we
// already use for staging dirs (.dumpsock-staging-*).
const sessionFileName = ".dumpsock-session.json"

// Session is the on-disk schema for an in-progress backup. Kept tiny
// and forward-compatible — adding fields is safe; rename or remove
// requires a Version bump and a migration path.
type Session struct {
	Version     int       `json:"version"`
	UDID        string    `json:"udid"`
	DeviceName  string    `json:"device_name,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	OutputRoot  string    `json:"output_root"`
	RemoteRoot  string    `json:"remote_root"`
	TotalJobs   int       `json:"total_jobs"`
	BytesTotal  int64     `json:"bytes_total"`
	Done        int       `json:"done"`
	BytesPulled int64     `json:"bytes_pulled"`
	CurrentFile string    `json:"current_file,omitempty"`
}

// WriteSession atomically writes (or overwrites) the session file in
// the given output dir. Atomic = write to a temp file then rename, so
// a crash mid-write never leaves a partial JSON document.
func WriteSession(outputDir string, s Session) error {
	if outputDir == "" {
		return errors.New("WriteSession: empty outputDir")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	s.UpdatedAt = time.Now()
	if s.Version == 0 {
		s.Version = 1
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	final := filepath.Join(outputDir, sessionFileName)
	tmp, err := os.CreateTemp(outputDir, sessionFileName+".tmp-*")
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

// ReadSession returns the session at outputDir, or (nil, nil) if no
// session file exists there. Returns a non-nil error only when the file
// exists but can't be read or parsed.
func ReadSession(outputDir string) (*Session, error) {
	path := filepath.Join(outputDir, sessionFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSession removes the sentinel. Called on clean completion of a
// pull (success OR explicit user cancellation — both are "the run is no
// longer in progress"). Idempotent: a missing file is not an error.
func DeleteSession(outputDir string) error {
	err := os.Remove(filepath.Join(outputDir, sessionFileName))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
