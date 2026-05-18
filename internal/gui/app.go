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
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/danielpaulus/go-ios/ios"
	goinstall "github.com/danielpaulus/go-ios/ios/installationproxy"

	"github.com/code-hartle-tech/dumpsock/internal/afc"
	"github.com/code-hartle-tech/dumpsock/internal/backup"
	"github.com/code-hartle-tech/dumpsock/internal/branding"
	"github.com/code-hartle-tech/dumpsock/internal/compare"
	"github.com/code-hartle-tech/dumpsock/internal/macauth"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the Wails bind target. JS sees its exported methods.
type App struct {
	ctx    context.Context
	mu     sync.Mutex
	job    *jobHandle
	jobSeq int
	cfg    Config
	// opCancel cancels whatever long-running operation is currently in
	// flight that isn't routed through a.job (decrypt, decrypt-and-
	// unarchive, etc.). Set by the binding before it kicks off the
	// op, cleared in defer. CancelCurrentOp uses this so the GUI's
	// single Cancel button works for every long path, not just pulls.
	opCancel context.CancelFunc
}

type jobHandle struct {
	id     string
	cancel context.CancelFunc
	done   chan struct{}
}

// NewApp constructs the App. Wails calls Startup(ctx) once the window is up.
func NewApp() *App { return &App{} }

// Startup is wired via Wails options.OnStartup. Loads persisted config
// (best-effort — failures fall back to zero-value config, never block)
// and discovers any pre-existing backup folders sitting in the default
// locations so the Backups tab populates immediately even for runs
// that predate the auto-register fix.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if c, err := loadConfig(); err == nil {
		a.mu.Lock()
		a.cfg = c
		a.mu.Unlock()
	}
	a.discoverBackups()
}

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
	Since       string `json:"since"` // YYYY-MM-DD; empty = no lower bound
	Until       string `json:"until"` // YYYY-MM-DD; empty = no upper bound
	Parallel    int    `json:"parallel"`
	UntilFound  int    `json:"until_found"`
	NoMtime     bool   `json:"no_mtime"`
	NoNotify    bool   `json:"no_notify"`
	DryRun      bool   `json:"dry_run"`
	DeleteAfter bool   `json:"delete_after"`
	ConfirmDel  bool   `json:"confirm_delete"`

	// Post-backup packaging (2026-05-17, expanded 2026-05-18).
	//   Compress=true → produce <outputRoot>/<leaf>.zip (Store mode).
	//   Password non-empty + PasswordMode="" or "standard" → encrypt
	//     each file in the zip with WinZip-AE-2 AES-256. Any zip tool
	//     with AES support (7-Zip, Keka, Finder, WinRAR) can decrypt.
	//   Password non-empty + PasswordMode="maximum" → above PLUS wrap
	//     the produced zip in DumpSock's DSAES2 container (AES-256-GCM
	//     PBKDF2-SHA256). Slower but only DumpSock decrypts the result.
	Compress     bool   `json:"compress"`
	Password     string `json:"password,omitempty"`
	PasswordMode string `json:"password_mode,omitempty"` // "" | "standard" | "maximum"

	// OnlyPaths, when non-empty, restricts the pull to exactly these
	// remote AFC paths (set from the Browse tab's checkbox selection).
	// When set, the engine skips the recursive DCIM walk entirely.
	OnlyPaths []string `json:"only_paths,omitempty"`

	// ArchiveScope controls what PackageZip bundles. "" or "all" →
	// everything under outputRoot (today's default; includes past
	// runs). "current" → only files res.PulledPaths from this run.
	ArchiveScope string `json:"archive_scope,omitempty"`

	// DeleteRawAfterArchive, when true AND archiving succeeded, walks
	// outputRoot and removes everything that ISN'T the produced
	// archive (.zip / .zip.aes) or the metadata sentinels
	// (.dumpsock.json / .dumpsock-session.json). The YYYY-MM-DD/
	// subfolders go away; the metadata + run history stays. Useful
	// when the user wants only the encrypted package on disk.
	DeleteRawAfterArchive bool `json:"delete_raw_after_archive,omitempty"`
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

// GetConfig returns the persisted user preferences. Reads fresh from
// disk on every call — sidesteps any race between Wails OnStartup
// (which sets a.cfg) and the JS bootstrap that calls this. The config
// file is tiny so the extra I/O is invisible.
func (a *App) GetConfig() Config {
	if c, err := loadConfig(); err == nil {
		a.mu.Lock()
		a.cfg = c
		a.mu.Unlock()
		return c
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// SaveLastOutput persists the folder the user just picked, so the next
// app launch restores it. Best-effort: errors are logged via fmt to
// stderr and surfaced to JS, but never block the pull flow.
func (a *App) SaveLastOutput(path string) error {
	a.mu.Lock()
	a.cfg.LastOutput = path
	// Also remember it in the KnownBackups list so it shows up in the
	// Backups tab. Dedup on insert.
	if path != "" {
		seen := false
		for _, p := range a.cfg.KnownBackups {
			if p == path {
				seen = true
				break
			}
		}
		if !seen {
			a.cfg.KnownBackups = append(a.cfg.KnownBackups, path)
		}
	}
	c := a.cfg
	a.mu.Unlock()
	if err := saveConfig(c); err != nil {
		return fmt.Errorf("could not persist last-output: %w", err)
	}
	return nil
}

// BackupEntry is one row in the Backups tab — combines the registry
// path with whatever .dumpsock.json metadata we can find at it.
//
// Archives lists any .zip / .zip.aes files sitting directly inside
// the backup folder (one level deep — we don't recurse the YYYY-MM-DD
// subtree). Drives the per-row "Decrypt" button on the Saved-backups
// list so the operator never has to drop to the CLI.
type BackupEntry struct {
	Path        string           `json:"path"`
	Reachable   bool             `json:"reachable"`
	Volume      string           `json:"volume,omitempty"`
	Metadata    *backup.Metadata `json:"metadata,omitempty"`
	Interrupted *backup.Session  `json:"interrupted,omitempty"`
	Archives    []string         `json:"archives,omitempty"`
}

// discoverBackups scans likely parent directories one level deep for
// folders containing a `.dumpsock.json` marker, plus checks LastOutput
// itself directly. Runs on Startup + on every ListBackups call so
// backups produced before the auto-register fix (2026-05-18) still
// appear and manually-copied backup folders are picked up.
//
// Candidate parents:
//   - ~/DumpSock         (the documented default root)
//   - ~/Documents        (common user choice, including operator's)
//   - ~/Desktop, ~/Downloads (also common)
//   - /Volumes/*         (external drives — each volume gets scanned)
//   - filepath.Dir(LastOutput) (catches whatever parent the user picked)
//
// Idempotent: existing KnownBackups entries are preserved; new ones
// are appended with dedup. Bounded — only the FIRST level beneath
// each candidate parent is scanned (no recursion). LastOutput itself
// is added directly when it contains a marker (covers the case where
// the operator picked it via Choose… and we want to capture exactly
// that path, not its siblings).
func (a *App) discoverBackups() {
	candidates := map[string]bool{}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates[filepath.Join(home, "DumpSock")] = true
		candidates[filepath.Join(home, "Documents")] = true
		candidates[filepath.Join(home, "Desktop")] = true
		candidates[filepath.Join(home, "Downloads")] = true
	}
	a.mu.Lock()
	lastOutput := a.cfg.LastOutput
	if lastOutput != "" {
		candidates[filepath.Dir(lastOutput)] = true
	}
	known := make(map[string]bool, len(a.cfg.KnownBackups))
	for _, p := range a.cfg.KnownBackups {
		known[p] = true
	}
	a.mu.Unlock()

	// Each mounted volume on macOS is a candidate root for "external
	// drive backups." Add /Volumes/<name> for each.
	if vols, err := os.ReadDir("/Volumes"); err == nil {
		for _, v := range vols {
			if v.IsDir() {
				candidates[filepath.Join("/Volumes", v.Name())] = true
			}
		}
	}

	var added []string
	// LastOutput itself, if it carries a marker, joins the list
	// directly (no parent scan needed).
	if lastOutput != "" && !known[lastOutput] {
		if _, err := os.Stat(filepath.Join(lastOutput, backup.MetadataFileName)); err == nil {
			added = append(added, lastOutput)
			known[lastOutput] = true
		} else {
			// Even without the marker, if the user picked this folder
			// recently it almost certainly IS a backup folder. Register
			// it conservatively — the row will show "no metadata" but
			// at least surfaces in the list.
			if fi, err := os.Stat(lastOutput); err == nil && fi.IsDir() {
				added = append(added, lastOutput)
				known[lastOutput] = true
			}
		}
	}

	for parent := range candidates {
		fi, err := os.Stat(parent)
		if err != nil || !fi.IsDir() {
			continue
		}
		entries, err := os.ReadDir(parent)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			full := filepath.Join(parent, e.Name())
			if known[full] {
				continue
			}
			if _, err := os.Stat(filepath.Join(full, backup.MetadataFileName)); err != nil {
				continue
			}
			added = append(added, full)
			known[full] = true
		}
	}
	if len(added) == 0 {
		return
	}
	a.mu.Lock()
	a.cfg.KnownBackups = append(a.cfg.KnownBackups, added...)
	cfg := a.cfg
	a.mu.Unlock()
	_ = saveConfig(cfg)
}

// ListBackups returns one BackupEntry per directory in KnownBackups,
// in newest-completion-first order. Entries whose path is unreachable
// (volume unplugged, directory deleted) still appear — the GUI shows
// them with a "plug the disk back in" hint.
func (a *App) ListBackups() ([]BackupEntry, error) {
	a.discoverBackups() // catch any pre-existing folders we haven't seen yet
	a.mu.Lock()
	paths := append([]string(nil), a.cfg.KnownBackups...)
	a.mu.Unlock()

	out := make([]BackupEntry, 0, len(paths))
	for _, p := range paths {
		entry := BackupEntry{Path: p}
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			entry.Reachable = true
			if md, _ := backup.ReadMetadata(p); md != nil {
				entry.Metadata = md
			}
			if s, _ := backup.ReadSession(p); s != nil {
				entry.Interrupted = s
			}
			// Detect any .zip / .zip.aes / .dumpsock sitting directly
			// inside the backup folder so the Backups tab can show a
			// Decrypt button. One level only — we don't recurse the
			// YYYY-MM-DD/ subtree. `.dumpsock.json` (metadata) is
			// excluded by the HasSuffix check — .json wins.
			if dir, derr := os.ReadDir(p); derr == nil {
				for _, e := range dir {
					if e.IsDir() {
						continue
					}
					name := e.Name()
					if strings.HasSuffix(name, ".dumpsock") ||
						strings.HasSuffix(name, ".zip.aes") ||
						strings.HasSuffix(name, ".zip") {
						entry.Archives = append(entry.Archives, filepath.Join(p, name))
					}
				}
			}
		}
		entry.Volume = volumeHint(p)
		out = append(out, entry)
	}
	// Sort: reachable first, then by most-recent update.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if !out[i].Reachable && out[j].Reachable {
				out[i], out[j] = out[j], out[i]
				continue
			}
			ti, tj := backupSortKey(out[i]), backupSortKey(out[j])
			if tj.After(ti) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// backupSortKey returns the timestamp used to order Backups list rows.
func backupSortKey(e BackupEntry) time.Time {
	if e.Metadata != nil {
		if !e.Metadata.UpdatedAt.IsZero() {
			return e.Metadata.UpdatedAt
		}
		return e.Metadata.CreatedAt
	}
	return time.Time{}
}

// volumeHint returns a short label identifying the disk a path lives on.
// macOS: "/Volumes/Lexar/..." → "Lexar". Used as a UX hint when the path
// is unreachable ("Plug Lexar back in"). Empty string when we can't tell.
func volumeHint(path string) string {
	if path == "" {
		return ""
	}
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "darwin" && strings.HasPrefix(cleaned, "/Volumes/") {
		rest := strings.TrimPrefix(cleaned, "/Volumes/")
		if i := strings.Index(rest, "/"); i > 0 {
			return rest[:i]
		}
		return rest
	}
	return ""
}

// MoveBackup relocates a backup directory from src to dst (which the
// user picks via a folder dialog). Updates the KnownBackups registry
// and rewrites the .dumpsock.json's OutputRoot to match the new path.
//
// First tries os.Rename — fast, atomic, works within one volume. Falls
// back to a recursive copy + remove when the rename fails (cross-disk
// move, ENOSPC, etc.).
func (a *App) MoveBackup(src, dst string) error {
	if src == "" || dst == "" {
		return errors.New("MoveBackup: src and dst required")
	}
	if src == dst {
		return errors.New("source and destination are the same")
	}
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("source unreachable: %w", err)
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir destination parent: %w", err)
	}

	// Fast path: same-volume rename.
	if err := os.Rename(src, dst); err != nil {
		// Cross-device or permission — fall back to copy + remove.
		if err := copyTree(src, dst); err != nil {
			_ = os.RemoveAll(dst) // tidy partial copy
			return fmt.Errorf("copy backup: %w", err)
		}
		if err := os.RemoveAll(src); err != nil {
			return fmt.Errorf("copy ok but source removal failed: %w", err)
		}
	}

	// Rewrite metadata so OutputRoot in .dumpsock.json reflects new home.
	if md, _ := backup.ReadMetadata(dst); md != nil {
		md.OutputRoot = dst
		_ = backup.WriteMetadata(dst, *md)
	}

	a.mu.Lock()
	for i, p := range a.cfg.KnownBackups {
		if p == src {
			a.cfg.KnownBackups[i] = dst
		}
	}
	if a.cfg.LastOutput == src {
		a.cfg.LastOutput = dst
	}
	c := a.cfg
	a.mu.Unlock()
	_ = saveConfig(c)
	return nil
}

// ForgetBackup drops a path from the KnownBackups registry. Does not
// touch the directory on disk — purely a UI/registry concept.
func (a *App) ForgetBackup(path string) error {
	a.mu.Lock()
	kept := a.cfg.KnownBackups[:0]
	for _, p := range a.cfg.KnownBackups {
		if p != path {
			kept = append(kept, p)
		}
	}
	a.cfg.KnownBackups = kept
	c := a.cfg
	a.mu.Unlock()
	return saveConfig(c)
}

// copyTree recursively copies src into dst (dst must not exist).
// Preserves file mode, mtime, and directory layout.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, _ := d.Info()
			mode := os.FileMode(0o755)
			if info != nil {
				mode = info.Mode().Perm()
			}
			return os.MkdirAll(target, mode)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		_ = os.Chtimes(target, info.ModTime(), info.ModTime())
		return nil
	})
}

// BrowseRemote returns the immediate children of an AFC path (one level,
// not recursive). Used by the Browse tab to render an iOS-Files-app-like
// view scoped to the com.apple.afc media jail (/, /DCIM, /Books, …).
//
// Opens a short-lived AFC connection per call — fine for browsing
// (latency is dominated by user clicks, not the connection setup).
func (a *App) BrowseRemote(udid, remotePath string) ([]afc.Entry, error) {
	if remotePath == "" {
		remotePath = "/"
	}
	cl, err := afc.Open(udid)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	return cl.List(remotePath)
}

// ExpandRemoteSelection takes a mixed list of file + folder remote
// paths and returns the flat list of regular-file entries beneath
// them. Used by the Browse tab to translate a folder-checkbox
// selection into the file-only list `StartBackup`'s `OnlyPaths` field
// expects.
//
// File paths come back unchanged; directory paths are recursively
// walked via the underlying AFC Walk (no extension filter — the user
// asked for the whole folder, we honor that).
func (a *App) ExpandRemoteSelection(udid string, paths []string) ([]afc.Entry, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	cl, err := afc.Open(udid)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	out := make([]afc.Entry, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		info, err := cl.Stat(p)
		if err != nil {
			// Skip unreadable entries — the GUI will surface the count
			// shortfall in its summary if any go missing.
			continue
		}
		if !info.IsDir() {
			if !seen[p] {
				out = append(out, afc.Entry{Name: filepath.Base(p), Path: p, Size: info.Size})
				seen[p] = true
			}
			continue
		}
		// Folder: walk it with no extension filter so the user gets
		// exactly what they checked, not just media types.
		files, err := cl.Walk(p, nil)
		if err != nil {
			continue
		}
		for _, f := range files {
			if !seen[f.Path] {
				out = append(out, afc.Entry{Name: f.Name, Path: f.Path, Size: f.Size})
				seen[f.Path] = true
			}
		}
	}
	return out, nil
}

// RemoteRemove deletes a path on the device — file or folder. Folders
// are removed recursively (RemoveAll). DESTRUCTIVE: there is no recycle
// bin on iOS. The frontend must show a confirm dialog before calling.
//
// Post-delete verification: AFC's remove returns success on iOS even
// when the file remains (iOS Photos reindex lag, NSFileProtectionClass
// holds, or post-iOS-15 permission cliffs on subtrees like /PhotoData).
// We re-Stat the path; if it still exists we surface that as an error
// so the frontend can show an honest "iOS refused" instead of a
// silent-failure "Deleted." toast.
func (a *App) RemoteRemove(udid, remotePath string) error {
	if remotePath == "" || remotePath == "/" {
		return errors.New("refusing to remove empty or root path")
	}
	cl, err := afc.Open(udid)
	if err != nil {
		return fmt.Errorf("AFC open: %w", err)
	}
	defer cl.Close()
	info, statErr := cl.Stat(remotePath)
	if statErr != nil {
		return fmt.Errorf("stat %s before delete: %w", remotePath, statErr)
	}
	wasDir := info.IsDir()
	var rmErr error
	if wasDir {
		rmErr = cl.RemoveAll(remotePath)
	} else {
		rmErr = cl.Remove(remotePath)
	}
	if rmErr != nil {
		return fmt.Errorf("AFC remove %s: %w", remotePath, rmErr)
	}
	// Verify. If the path still stats, iOS / AFC kept the file despite
	// returning success — surface this honestly.
	if _, postErr := cl.Stat(remotePath); postErr == nil {
		return fmt.Errorf("iOS held onto %s (AFC reported success but path still exists — common on /PhotoData subtrees and Photos.app-managed entries; deletion from inside Photos is the workaround)", remotePath)
	}
	return nil
}

// FileSharingApp is one row in the "Browse → Apps" picker. Only apps
// that ship `UIFileSharingEnabled=YES` in their Info.plist appear here
// (DJI Fly, VLC, Procreate, Documents, GarageBand, etc.) — banking,
// messaging, and system apps never publish the flag and aren't reachable
// via House Arrest on a non-jailbroken phone.
type FileSharingApp struct {
	BundleID   string `json:"bundle_id"`
	Name       string `json:"name"`
	Executable string `json:"executable,omitempty"`
	Version    string `json:"version,omitempty"`
}

// ListFileSharingApps enumerates third-party (App-Store-installed)
// apps whose Info.plist has UIFileSharingEnabled=YES. The Browse tab
// uses this to populate the "App data" picker.
//
// Why two filters: go-ios v1.0.213's `BrowseFileSharingApps` doesn't
// actually filter the device-side response (the agent report claimed
// otherwise; the source proves it sends `{Command: "Browse"}` with no
// filter). Even after a host-side UIFileSharingEnabled check, Apple
// system apps like com.apple.Fitness still appear — they declare the
// flag but house_arrest refuses to vend them to third parties, so
// every attempt fails with EOF or InstallationLookupFailed. Using
// `BrowseUserApps` (ApplicationType=User on the device side) drops
// system apps first; the host-side UIFileSharingEnabled check then
// narrows to the actually-reachable subset.
func (a *App) ListFileSharingApps(udid string) ([]FileSharingApp, error) {
	dev, err := pickDevice(udid)
	if err != nil {
		return nil, err
	}
	conn, err := goinstall.New(dev)
	if err != nil {
		return nil, fmt.Errorf("installation_proxy: %w", err)
	}
	defer conn.Close()
	apps, err := conn.BrowseUserApps()
	if err != nil {
		return nil, fmt.Errorf("browse user apps: %w", err)
	}
	out := make([]FileSharingApp, 0, len(apps))
	for _, ai := range apps {
		if !ai.UIFileSharingEnabled() {
			continue
		}
		out = append(out, FileSharingApp{
			BundleID:   ai.CFBundleIdentifier(),
			Name:       ai.CFBundleName(),
			Executable: ai.CFBundleExecutable(),
			Version:    ai.CFBundleShortVersionString(),
		})
	}
	return out, nil
}

// BrowseApp lists a single level inside a third-party app's data
// container via the com.apple.mobile.house_arrest lockdownd service.
// `relPath` is relative to the app's vended root (use "/" to start).
//
// Routes through afc.OpenAppContainer, which tries VendDocuments first
// (works on production-signed apps with UIFileSharingEnabled=YES) and
// falls back to VendContainer (dev-signed). Replaces an earlier
// gohouse.New call that hardcoded VendContainer and failed with
// InstallationLookupFailed on real-world App-Store apps like DJI GO
// Lite (com.dji.golite).
func (a *App) BrowseApp(udid, bundleID, relPath string) ([]afc.Entry, error) {
	if bundleID == "" {
		return nil, errors.New("BrowseApp: empty bundleID")
	}
	if relPath == "" {
		relPath = "/"
	}
	cl, err := afc.OpenAppContainer(udid, bundleID)
	if err != nil {
		return nil, err
	}
	defer cl.Close()
	return cl.List(relPath)
}

// PickArchiveToDecrypt pops a native file-open dialog scoped to
// DumpSock-encrypted archives. Returns the chosen path, or "" when the
// user cancels. Used by the Backups tab's "Decrypt an archive…" button
// so the entire flow stays inside the GUI (operator preference 2026-
// 05-18: "everything stays in the UI, don't be lazy").
//
// Filter spec note: only single-pattern entries are used (no
// semicolon-separated multi-ext) because operator hit a force-close on
// the previous build using `*.aes;*.zip.aes`; some Wails/macOS combos
// crash on that syntax. A `.zip.aes` file matches `*.aes` anyway.
func (a *App) PickArchiveToDecrypt() (string, error) {
	return wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Pick a DumpSock-encrypted archive",
		Filters: []wruntime.FileFilter{
			{DisplayName: "DumpSock encrypted (*.dumpsock)", Pattern: "*.dumpsock"},
			{DisplayName: "Legacy encrypted (*.aes)", Pattern: "*.aes"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
}

// DecryptArchive runs backup.DecryptFile on src using password. If
// outPath is empty we pick <src-without-.aes>, auto-suffixing -1, -2,
// … if that name is already taken (so the operator can re-decrypt
// without manually clearing the prior output). Returns the produced
// path on success. Forwards decrypt progress to the GUI via the same
// backup:progress event stream the pull/pack/encrypt phases use.
func (a *App) DecryptArchive(srcPath, outPath, password string) (string, error) {
	if srcPath == "" {
		return "", errors.New("DecryptArchive: empty srcPath")
	}
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("source not readable: %w", err)
	}
	if outPath == "" {
		// Default decrypted output name. Strip the brand extension
		// (.dumpsock) or legacy (.aes) and ensure a .zip suffix.
		base := strings.TrimSuffix(srcPath, ".dumpsock")
		if base == srcPath {
			base = strings.TrimSuffix(srcPath, ".aes")
		}
		if !strings.HasSuffix(base, ".zip") {
			base += ".zip"
		}
		outPath = base
		if outPath == srcPath {
			outPath = srcPath + ".decrypted"
		}
	}
	// Auto-suffix when the chosen name already exists. Prior versions
	// failed loudly with "output already exists" — operator-reported
	// friction 2026-05-18.
	if _, err := os.Stat(outPath); err == nil {
		outPath = nextFreeName(outPath)
	}
	onProgress := func(pp backup.PackageProgress) {
		wruntime.EventsEmit(a.ctx, "backup:progress", backup.ProgressEvent{
			Phase:       pp.Phase, // "decrypting"
			BytesTotal:  pp.BytesTotal,
			BytesPulled: pp.BytesDone,
			Current:     pp.Current,
		})
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock(); a.opCancel = cancel; a.mu.Unlock()
	defer func() { a.mu.Lock(); a.opCancel = nil; a.mu.Unlock(); cancel() }()
	return backup.DecryptFile(ctx, srcPath, outPath, password, onProgress)
}

// DecryptAndExtractArchive does the full "decrypt → unzip → drop the
// intermediate .zip → return the extracted folder" pipeline so the
// operator can go from .zip.aes to a browsable folder in one click.
// Surfaced next to the existing Reveal-archive / Last-Backup actions
// (operator preference 2026-05-18 "there should be a decrypt &
// unarchive next to last backup & reveal").
//
// Progress emits backup:progress events with two phases:
//   - "decrypting" — DSAES2 chunk decrypt
//   - "extracting" — zip entries unpacked
// The frontend's existing byte-bar handles both.
func (a *App) DecryptAndExtractArchive(srcPath, password string) (string, error) {
	if srcPath == "" {
		return "", errors.New("DecryptAndExtractArchive: empty srcPath")
	}
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("source not readable: %w", err)
	}
	// 1) Decrypt to a sibling .zip (auto-suffix if taken). Handles
	// both new .dumpsock files and the legacy .aes naming.
	base := strings.TrimSuffix(srcPath, ".dumpsock")
	if base == srcPath {
		base = strings.TrimSuffix(srcPath, ".aes")
	}
	if !strings.HasSuffix(base, ".zip") {
		base += ".zip"
	}
	intermediateZip := base
	if intermediateZip == srcPath {
		intermediateZip = srcPath + ".decrypted.zip"
	}
	if _, err := os.Stat(intermediateZip); err == nil {
		intermediateZip = nextFreeName(intermediateZip)
	}
	progress := func(pp backup.PackageProgress) {
		wruntime.EventsEmit(a.ctx, "backup:progress", backup.ProgressEvent{
			Phase:       pp.Phase, // "decrypting" or "extracting"
			BytesTotal:  pp.BytesTotal,
			BytesPulled: pp.BytesDone,
			Current:     pp.Current,
		})
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.mu.Lock(); a.opCancel = cancel; a.mu.Unlock()
	defer func() { a.mu.Lock(); a.opCancel = nil; a.mu.Unlock(); cancel() }()
	if _, err := backup.DecryptFile(ctx, srcPath, intermediateZip, password, progress); err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	// 2) Extract to a sibling folder. If the extract fails or is
	// cancelled, ExtractZip's own deferred cleanup removes the
	// partial directory so the operator doesn't have to.
	dstDir := strings.TrimSuffix(intermediateZip, ".zip")
	if dstDir == intermediateZip {
		dstDir = intermediateZip + ".extracted"
	}
	if _, err := os.Stat(dstDir); err == nil {
		dstDir = nextFreeName(dstDir)
	}
	// Pass the SAME password to ExtractZip — Maximum mode's inner zip
	// has per-file AES-256 encryption (Standard layer); the DSAES2
	// wrapper sits outside that. Both layers use the same password.
	// Operator-reported failure 2026-05-18 where the extract step
	// errored with "wrong password" even though the password was
	// correct — bug was passing "" here.
	if _, err := backup.ExtractZip(ctx, intermediateZip, dstDir, password, progress); err != nil {
		// Leave the .zip in place so the operator can retry without
		// re-decrypting the (slow) outer container.
		return "", fmt.Errorf("extract: %w (decrypted zip kept at %s)", err, intermediateZip)
	}
	// 3) Drop the intermediate zip — extraction succeeded.
	_ = os.Remove(intermediateZip)
	return dstDir, nil
}

// removeRawTree walks outputRoot one level deep and deletes every
// entry that ISN'T a known DumpSock artifact (the .zip / .zip.aes
// archive or the .dumpsock.* metadata sentinels). YYYY-MM-DD/
// subfolders and any stray files get removed recursively. The
// .dumpsock.json history stays so Saved Backups still surfaces the
// folder with its run record. Defensive: bails on empty outputRoot
// or any unstatable path.
func removeRawTree(outputRoot string) error {
	if outputRoot == "" || outputRoot == "/" {
		return errors.New("removeRawTree: refusing empty or root path")
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		// Preserve archives + metadata + sentinels. Everything else
		// (YYYY-MM-DD/ subfolders, the "0000:00:00 00:00:00" no-date
		// bucket, stray files) goes.
		if strings.HasPrefix(name, ".dumpsock") ||
			strings.HasSuffix(name, ".zip") ||
			strings.HasSuffix(name, ".zip.aes") ||
			strings.HasSuffix(name, ".dumpsock") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(outputRoot, name))
	}
	return nil
}

// CancelCurrentOp cancels whatever long-running operation is in flight:
// pull (via the legacy a.job.cancel), decrypt, decrypt-and-unarchive,
// extraction. Safe to call when nothing's running.
func (a *App) CancelCurrentOp() {
	a.mu.Lock()
	op := a.opCancel
	job := a.job
	a.mu.Unlock()
	if op != nil {
		op()
	}
	if job != nil {
		job.cancel()
	}
}

// nextFreeName returns base if it doesn't exist, otherwise base with
// -1, -2, … suffix inserted before the extension. Bounded at 1000
// attempts. Mirrors backup.freeName but isn't exported there; kept
// inline here to avoid widening the package API for a one-call use.
func nextFreeName(base string) string {
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

// BiometricsAvailable reports whether the OS can present a Touch ID
// (or login-password fallback) prompt. Linux/Windows builds always
// return false. The Settings UI greys the "Use Touch ID" toggle when
// this is false.
func (a *App) BiometricsAvailable() bool {
	return macauth.Available()
}

// HasBiometricPassword reports whether the user has already enrolled a
// backup password behind Touch ID. Cheap, doesn't prompt — the UI
// calls this on Settings-tab open.
func (a *App) HasBiometricPassword() bool {
	return macauth.HasPassword()
}

// EnrollBiometricPassword stores `password` in Keychain behind a
// biometry-current-set access control. Future LoadBiometricPassword
// calls will present a Touch ID sheet. Idempotent — overwrites any
// existing entry.
func (a *App) EnrollBiometricPassword(password string) error {
	return macauth.StorePassword(password)
}

// LoadBiometricPassword presents the Touch ID sheet and returns the
// stored password. The system blocks the calling goroutine while the
// sheet is up — Wails bindings run off the UI thread, so this is fine.
func (a *App) LoadBiometricPassword() (string, error) {
	return macauth.LoadPassword("Unlock your DumpSock backup password")
}

// ClearBiometricPassword wipes the stored entry. Doesn't prompt.
func (a *App) ClearBiometricPassword() error {
	return macauth.ClearPassword()
}

// pickDevice resolves a UDID to a go-ios DeviceEntry. Empty UDID picks
// the only attached device. (Shared with afc.pickDevice in spirit but
// re-implemented here to keep package boundaries clean.)
func pickDevice(udid string) (ios.DeviceEntry, error) {
	list, err := ios.ListDevices()
	if err != nil {
		return ios.DeviceEntry{}, fmt.Errorf("usbmuxd: %w", err)
	}
	if len(list.DeviceList) == 0 {
		return ios.DeviceEntry{}, errors.New("no iPhone connected over USB")
	}
	if udid == "" {
		if len(list.DeviceList) > 1 {
			return ios.DeviceEntry{}, errors.New("multiple devices — pass --udid")
		}
		return list.DeviceList[0], nil
	}
	for _, d := range list.DeviceList {
		if d.Properties.SerialNumber == udid {
			return d, nil
		}
	}
	return ios.DeviceEntry{}, fmt.Errorf("device %s not connected", udid)
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

// Storage is the iPhone storage snapshot exposed to the GUI's storage gauge.
type Storage struct {
	Model      string `json:"model"`
	TotalBytes uint64 `json:"total_bytes"`
	FreeBytes  uint64 `json:"free_bytes"`
	UsedBytes  uint64 `json:"used_bytes"`
}

// DeviceStorage queries AFC for the connected device's storage capacity.
// Returns a zero-valued Storage with an error if no device is reachable.
func (a *App) DeviceStorage(udid string) (Storage, error) {
	cl, err := afc.Open(udid)
	if err != nil {
		return Storage{}, err
	}
	defer cl.Close()
	s, err := cl.DeviceStorage()
	if err != nil {
		return Storage{}, err
	}
	used := uint64(0)
	if s.TotalBytes > s.FreeBytes {
		used = s.TotalBytes - s.FreeBytes
	}
	return Storage{
		Model:      s.Model,
		TotalBytes: s.TotalBytes,
		FreeBytes:  s.FreeBytes,
		UsedBytes:  used,
	}, nil
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
			OnlyPaths:   req.OnlyPaths,
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

		// Auto-register the outputRoot in KnownBackups so the Backups
		// tab's saved-list populates without requiring the user to pick
		// a "new" folder via Choose…. Operator-reported fix 2026-05-18.
		// Only on successful (or partially-successful) runs — failures
		// don't make a folder a "saved backup."
		if err == nil && !req.DryRun {
			a.mu.Lock()
			seen := false
			for _, p := range a.cfg.KnownBackups {
				if p == opts.OutputRoot {
					seen = true
					break
				}
			}
			if !seen && opts.OutputRoot != "" {
				a.cfg.KnownBackups = append(a.cfg.KnownBackups, opts.OutputRoot)
				cfg := a.cfg
				a.mu.Unlock()
				_ = saveConfig(cfg)
			} else {
				a.mu.Unlock()
			}
		}

		// Post-backup packaging. Compress=true → produce
		// <outputRoot>/<leaf>.zip; Password non-empty → additionally
		// encrypt to <leaf>.zip.aes and delete the intermediate plain
		// zip. Skipped on error/dryrun since those don't write a
		// complete tree. Progress is forwarded via backup:progress
		// events so the GUI's progress bar stays live during packaging
		// (a multi-GB backup otherwise looks like a freeze).
		if err == nil && !req.DryRun && (req.Compress || req.Password != "") {
			packProgress := func(pp backup.PackageProgress) {
				wruntime.EventsEmit(a.ctx, "backup:progress", backup.ProgressEvent{
					Phase:       pp.Phase,
					Current:     pp.Current,
					BytesTotal:  pp.BytesTotal,
					BytesPulled: pp.BytesDone,
				})
			}
			mode := backup.PasswordMode(req.PasswordMode)
			if req.Password == "" {
				mode = backup.PasswordNone
			} else if mode == "" {
				mode = backup.PasswordStandard
			}
			// Archive scope: default is "all" (walk outputRoot, includes
			// past runs), or "current" (only this run's freshly-pulled
			// files via res.PulledPaths). If "current" but nothing was
			// pulled, we surface that as the package_error instead of
			// silently producing an empty archive.
			var scopedPaths []string
			scopeErr := ""
			if req.ArchiveScope == "current" {
				scopedPaths = res.PulledPaths
				if len(scopedPaths) == 0 {
					scopeErr = "Only-this-run archive requested but no files were pulled — nothing to bundle."
				}
			}
			if scopeErr != "" {
				payload["package_error"] = scopeErr
			} else {
				zipPath, pErr := backup.PackageZip(ctx, opts.OutputRoot, req.Password, mode, scopedPaths, packProgress)
				if pErr != nil {
					payload["package_error"] = pErr.Error()
				} else {
					payload["zip_path"] = zipPath
					// Maximum mode adds the DSAES2 outer wrapper on top
					// of the password-encrypted zip; only DumpSock
					// decrypts the result.
					if mode == backup.PasswordMaximum {
						encPath, eErr := backup.EncryptFile(ctx, zipPath, req.Password, packProgress)
						if eErr != nil {
							payload["package_error"] = eErr.Error()
						} else {
							_ = os.Remove(zipPath) // intermediate inner zip
							payload["encrypted_path"] = encPath
						}
					}
				}
			}

			// Optional: remove the raw YYYY-MM-DD/ tree once archiving
			// succeeded. Operator preference 2026-05-18 — useful when
			// the user wants only the encrypted package on disk. The
			// .dumpsock.json + RunRecord history stays so Saved Backups
			// still shows what was archived. Only fires when packaging
			// reported no error — never half-deletes.
			if req.DeleteRawAfterArchive && payload["package_error"] == nil {
				_ = removeRawTree(opts.OutputRoot)
			}
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

// RunCompare powers the Compare & Merge tab. Walks the device's DCIM via
// AFC and the destination folder tree, categorizes by media type, and
// returns the per-side breakdown plus delta counts.
//
// Safe to call concurrently with a backup — it opens its own AFC client
// since the backup engine holds its own client and AFC sessions are
// cheap enough that double-opening is fine for a comparison run.
//
// Emits "compare:progress" events with phase strings (walking_device,
// walking_backup, categorizing, diffing, done) so the UI can show what
// the comparison is doing on a big device.
func (a *App) RunCompare(udid, outputRoot string) (compare.Result, error) {
	if outputRoot == "" {
		return compare.Result{}, errors.New("no destination folder set yet")
	}
	cl, err := afc.Open(udid)
	if err != nil {
		return compare.Result{}, err
	}
	defer func() { _ = cl.Close() }()

	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	res, err := compare.Compute(ctx, compare.Options{
		Conn:      cl,
		LocalRoot: outputRoot,
		OnProgress: func(p compare.Progress) {
			wruntime.EventsEmit(a.ctx, "compare:progress", p)
		},
	})
	if err != nil {
		return compare.Result{}, err
	}
	return res, nil
}

// PathExists reports whether the given filesystem path is currently
// reachable. Used by the GUI to validate a remembered output folder
// before kicking off a backup — if the saved Lexar SSD isn't plugged in
// or the directory got deleted, we want to open the picker rather than
// fail mid-pull.
func (a *App) PathExists(p string) bool {
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// GetInterruptedSession returns the .dumpsock-session.json sentinel
// from outputDir if one is present (a previous backup that didn't
// complete cleanly), or nil if the directory is in a quiescent state.
// The frontend calls this on launch and shows a "your last run was
// interrupted — resume?" prompt when something comes back.
//
// Resuming is functionally identical to re-running the pull at the
// same output dir: dedup will skip everything already on disk, then
// pull the remaining jobs. We rely on that property rather than
// trying to replay a partial work queue.
func (a *App) GetInterruptedSession(outputDir string) (*backup.Session, error) {
	if outputDir == "" {
		return nil, nil
	}
	return backup.ReadSession(outputDir)
}

// RevealDeviceInFinder brings Finder forward at a known-good location
// so the user can click their iPhone in the sidebar's Locations section.
//
// Why not auto-select the iPhone? Finder's scripting dictionary has no
// `sidebar` property, so `select first item of (sidebar of window 1)`
// silently fails. The only working path is System Events GUI-scripting,
// which requires the user to grant Accessibility permission via a TCC
// prompt — disproportionate UX cost for a small convenience. (Earlier
// versions tried the AppleScript route and ended up opening Finder at
// "the default New Finder Window target", which for many users is the
// folder where DumpSock.app lives — confusing.)
//
// We instead open Finder at the user's home folder and rely on a toast
// in the frontend to tell the user where to look. Returns false so the
// frontend always uses the "click your iPhone in the sidebar" copy.
func (a *App) RevealDeviceInFinder(deviceName string) (bool, error) {
	_ = deviceName // kept in the signature so frontend doesn't need a binding change
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "/"
		}
		_ = exec.Command("open", home).Start()
		return false, nil
	case "linux":
		_ = exec.Command("xdg-open", ".").Start()
		return false, nil
	case "windows":
		_ = exec.Command("explorer", ".").Start()
		return false, nil
	}
	return false, fmt.Errorf("RevealDeviceInFinder not implemented for %s", runtime.GOOS)
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

// RevealFileInFinder opens the parent directory AND selects the file
// itself. Used by the "Reveal archive" button on the done banner so
// the user lands directly on the .zip/.zip.aes (which sits as a
// sibling to the backup folder, not inside it — a confusing surprise
// the operator hit on 2026-05-18).
//
// macOS: `open -R <file>` selects it in Finder. Linux/Windows: open
// the parent directory (best effort; native shells don't have a
// universal "select this file" verb).
func (a *App) RevealFileInFinder(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path not accessible: %w", err)
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "linux":
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	case "windows":
		// /select arg selects the file in Explorer.
		cmd = exec.Command("explorer", "/select,", path)
	default:
		return fmt.Errorf("RevealFileInFinder not implemented for %s", runtime.GOOS)
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
