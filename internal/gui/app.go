// Package gui hosts the Wails-bound App type. Methods on App are
// callable from the embedded frontend via the Wails runtime; progress is
// pushed via runtime.EventsEmit.
//
// Concurrency contract: at most one in-flight backup per App instance.
package gui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/danielpaulus/go-ios/ios"

	"github.com/code-hartle-tech/dumpsock/internal/backup"
	"github.com/code-hartle-tech/dumpsock/internal/branding"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails bind target. JS sees its exported methods.
type App struct {
	ctx    context.Context
	mu     sync.Mutex
	job    *jobHandle
	jobSeq int
}

type jobHandle struct {
	id     string
	cancel context.CancelFunc
	done   chan struct{}
}

// NewApp constructs the App. Wails calls Startup(ctx) once the window is up.
func NewApp() *App { return &App{} }

// Startup is wired via Wails options.OnStartup.
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

// -----------------------------------------------------------------------------
// JS-visible types
// -----------------------------------------------------------------------------

// Device is the GUI's view of one connected iPhone.
type Device struct {
	UDID           string `json:"udid"`
	Name           string `json:"name"`
	ProductType    string `json:"product_type"`
	ProductVersion string `json:"product_version"`
	ConnectionType string `json:"connection_type"`
}

// BackupRequest is what the frontend posts to StartBackup.
type BackupRequest struct {
	UDID        string `json:"udid"`
	OutputRoot  string `json:"output_root"`
	Since       string `json:"since"`  // YYYY-MM-DD; empty = no lower bound
	Until       string `json:"until"`  // YYYY-MM-DD; empty = no upper bound
	Parallel    int    `json:"parallel"`
	UntilFound  int    `json:"until_found"`
	NoMtime     bool   `json:"no_mtime"`
	NoNotify    bool   `json:"no_notify"`
	DryRun      bool   `json:"dry_run"`
	DeleteAfter bool   `json:"delete_after"`
	ConfirmDel  bool   `json:"confirm_delete"`
}

// AppInfo carries static metadata to the UI for headers / about dialog.
type AppInfo struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Brand    string `json:"brand"`
	Contact  string `json:"contact"`
	Tagline  string `json:"tagline"`
	Platform string `json:"platform"`
}

// -----------------------------------------------------------------------------
// JS-visible methods
// -----------------------------------------------------------------------------

// Info returns the static brand block used in the UI header / about.
func (a *App) Info() AppInfo {
	return AppInfo{
		Name:     "DumpSock",
		Version:  branding.Version,
		Commit:   branding.Commit,
		Brand:    "HARTLE.TECH",
		Contact:  "contact@hartle.tech",
		Tagline:  branding.Tagline,
		Platform: runtime.GOOS + "/" + runtime.GOARCH,
	}
}

// ListDevices enumerates iPhones reachable via usbmuxd. JS calls this on
// the connect screen and refreshes when the user clicks "rescan".
func (a *App) ListDevices() ([]Device, error) {
	list, err := ios.ListDevices()
	if err != nil {
		return nil, fmt.Errorf("usbmuxd unreachable: %w", err)
	}
	out := make([]Device, 0, len(list.DeviceList))
	for _, e := range list.DeviceList {
		udid := e.Properties.SerialNumber
		d := Device{
			UDID:           udid,
			ConnectionType: e.Properties.ConnectionType,
		}
		if dev, err := ios.GetDevice(udid); err == nil {
			if v, err := ios.GetValues(dev); err == nil {
				d.Name = v.Value.DeviceName
				d.ProductType = v.Value.ProductType
				d.ProductVersion = v.Value.ProductVersion
			}
		}
		out = append(out, d)
	}
	return out, nil
}

// DefaultOutputFor proposes a default output directory for the given
// device name. Used to pre-fill the output picker.
func (a *App) DefaultOutputFor(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if name == "" {
		name = "iPhone"
	}
	return filepath.Join(home, "DumpSock", name)
}

// PickDirectory opens a native folder-chooser. Returns the absolute path
// or empty if the user cancels.
func (a *App) PickDirectory(title string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("app not started")
	}
	if title == "" {
		title = "Choose output folder"
	}
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: title,
	})
}

// StartBackup kicks off a backup goroutine and returns its job ID. The
// goroutine emits "backup:progress" and "backup:done" events; the UI
// subscribes via window.runtime.EventsOn.
func (a *App) StartBackup(req BackupRequest) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.job != nil {
		return "", errors.New("a backup is already running")
	}
	since, err := parseDate(req.Since, "since")
	if err != nil {
		return "", err
	}
	until, err := parseDate(req.Until, "until")
	if err != nil {
		return "", err
	}
	if req.DeleteAfter && !req.ConfirmDel {
		return "", errors.New("delete-after requires explicit confirmation")
	}
	a.jobSeq++
	id := fmt.Sprintf("job-%d", a.jobSeq)
	ctx, cancel := context.WithCancel(a.ctx)
	a.job = &jobHandle{id: id, cancel: cancel, done: make(chan struct{})}

	go func() {
		defer close(a.job.done)
		opts := backup.Options{
			UDID:        req.UDID,
			OutputRoot:  req.OutputRoot,
			Since:       since,
			Until:       until,
			Parallel:    nonZero(req.Parallel, 4),
			UntilFound:  req.UntilFound,
			SetMtime:    !req.NoMtime,
			Notify:      !req.NoNotify,
			DryRun:      req.DryRun,
			DeleteAfter: req.DeleteAfter,
			Out:         &eventWriter{ctx: a.ctx, event: "backup:log"},
			Err:         &eventWriter{ctx: a.ctx, event: "backup:log"},
			OnProgress: func(ev backup.ProgressEvent) {
				wruntime.EventsEmit(a.ctx, "backup:progress", ev)
			},
		}
		res, err := backup.Run(ctx, opts)
		payload := map[string]any{"result": res}
		if err != nil {
			payload["error"] = err.Error()
		}
		wruntime.EventsEmit(a.ctx, "backup:done", payload)

		a.mu.Lock()
		a.job = nil
		a.mu.Unlock()
	}()

	return id, nil
}

// CancelBackup signals the in-flight backup to stop. Safe to call when no
// job is running.
func (a *App) CancelBackup() {
	a.mu.Lock()
	job := a.job
	a.mu.Unlock()
	if job != nil {
		job.cancel()
	}
}

// RevealInFinder opens the destination folder in the platform's native
// file browser. Despite the macOS-flavored name, it works on all three
// supported platforms — name kept for the GUI button label.
//
// Semantics: opens the folder so its contents are visible. (Not "reveal"
// in the strict macOS sense of selecting the path in its parent.)
func (a *App) RevealInFinder(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path not accessible: %w", err)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		return fmt.Errorf("RevealInFinder not implemented for %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	return nil
}

// -----------------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------------

func parseDate(s, name string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: expected YYYY-MM-DD, got %q", name, s)
	}
	return t, nil
}

func nonZero(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}

// eventWriter satisfies io.Writer and emits each line as a Wails event so
// the frontend's log pane sees the same text the CLI prints.
type eventWriter struct {
	ctx   context.Context
	event string
}

func (w *eventWriter) Write(p []byte) (int, error) {
	wruntime.EventsEmit(w.ctx, w.event, string(p))
	return len(p), nil
}
