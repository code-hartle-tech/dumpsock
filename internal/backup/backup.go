// Package backup is the DumpSock pull orchestrator. It glues AFC, EXIF,
// dedup, notification, and file I/O into one Run() invocation.
package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/code-hartle-tech/dumpsock/internal/afc"
	"github.com/code-hartle-tech/dumpsock/internal/dedup"
	"github.com/code-hartle-tech/dumpsock/internal/exif"
	"github.com/code-hartle-tech/dumpsock/internal/notify"
)

// MediaExts is the default set of file extensions DumpSock will pull from
// DCIM. Case-insensitive; leading dot.
var MediaExts = map[string]bool{
	".heic": true, ".heif": true,
	".jpg": true, ".jpeg": true, ".png": true,
	".mov": true, ".mp4": true, ".m4v": true,
	".dng": true, ".raw": true,
	".gif": true, ".webp": true,
}

// SidecarExts are file types we never pull but DO delete alongside their
// media sibling when --delete-after is set. .AAE is Apple's photo-edit
// history metadata (crops, filters, rotation); a stray .AAE without its
// parent .HEIC/.MOV is orphan junk on the device, so we sweep it.
var SidecarExts = map[string]bool{".aae": true}

func extUnion(maps ...map[string]bool) map[string]bool {
	out := map[string]bool{}
	for _, m := range maps {
		for k := range m {
			out[k] = true
		}
	}
	return out
}

func stemPath(p string) string {
	return p[:len(p)-len(filepath.Ext(p))]
}

const noDateFolder = "0000:00:00 00:00:00"

// ProgressEvent is emitted at the same moments as the CLI's progress
// lines, so a GUI or a JSON consumer can subscribe to structured updates
// instead of parsing text. Phase is one of: "indexing", "walking",
// "planning", "pulling", "deleting", "done", "error".
type ProgressEvent struct {
	Phase        string `json:"phase"`
	Message      string `json:"message,omitempty"`
	Done         int    `json:"done,omitempty"`
	Total        int    `json:"total,omitempty"`
	Current      string `json:"current,omitempty"`
	Pulled       int    `json:"pulled,omitempty"`
	PreSkipped   int    `json:"pre_skipped,omitempty"`
	PostSkipped  int    `json:"post_skipped,omitempty"`
	Suffixed     int    `json:"suffixed,omitempty"`
	NoDate       int    `json:"nodate,omitempty"`
	Filtered     int    `json:"filtered,omitempty"`
	Errors       int    `json:"errors,omitempty"`
	Deleted      int    `json:"deleted,omitempty"`
	DeleteErrors int    `json:"delete_errors,omitempty"`
	BytesPulled  int64  `json:"bytes_pulled,omitempty"`
}

// Options drives a single Run() invocation. Most fields map 1:1 to the
// CLI flags defined in cmd/dumpsock; see CLAUDE.md "CLI surface".
type Options struct {
	UDID        string
	OutputRoot  string // empty => derive from device name
	RemoteRoot  string // default "DCIM"
	Since       time.Time
	Until       time.Time
	DeleteAfter bool
	DryRun      bool

	Parallel   int  // default 4
	UntilFound int  // 0 = disabled
	SetMtime   bool // default true (CLI: --no-mtime flips this)
	Notify     bool // default true (CLI: --no-notify flips this)
	JSONStream bool

	// Out / Err sinks for human-readable progress. JSON stream (if enabled)
	// also goes to Out.
	Out io.Writer
	Err io.Writer

	// OnProgress, when non-nil, is invoked at the same moments as the
	// human-readable progress lines. Used by the GUI (and any JSON
	// consumer) to surface structured progress. Called from arbitrary
	// goroutines — must be concurrency-safe.
	OnProgress func(ProgressEvent)
}

func (o *Options) emit(ev ProgressEvent) {
	if o.OnProgress != nil {
		o.OnProgress(ev)
	}
}

// Result summarizes one Run().
type Result struct {
	Total          int
	PreSkipped     int // name+size already at destination, never pulled
	Pulled         int
	PostSkipped    int // pulled, then discovered destination duplicate
	Suffixed       int // landed with -N suffix due to name collision
	NoDate         int // landed in 0000:00:00 00:00:00/
	Filtered       int // dropped by --since / --until
	Errors         int
	Deleted        int // remote files removed from device after verified copy
	DeleteErrors   int // remote files we tried to delete but couldn't
	UntilFoundStop bool // true if we early-exited via --until-found N
	Elapsed        time.Duration
}

// Run executes one pull pass. Watch-mode is implemented by the caller
// looping on Run().
func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.RemoteRoot == "" {
		opts.RemoteRoot = "DCIM"
	}
	if opts.Parallel <= 0 {
		opts.Parallel = 4
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	exr, err := exif.NewReader()
	if err != nil {
		return Result{}, fmt.Errorf("exif reader: %w", err)
	}

	cl, err := afc.Open(opts.UDID)
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = cl.Close() }()

	if opts.OutputRoot == "" {
		opts.OutputRoot, err = defaultOutput(cl.DeviceName())
		if err != nil {
			return Result{}, err
		}
	}
	if err := os.MkdirAll(opts.OutputRoot, 0o755); err != nil {
		return Result{}, fmt.Errorf("mkdir output: %w", err)
	}

	opts.emit(ProgressEvent{Phase: "indexing", Message: opts.OutputRoot})
	fmt.Fprintf(opts.Out, "indexing existing files under %s\n", opts.OutputRoot)
	ix, err := dedup.NewIndex(opts.OutputRoot)
	if err != nil {
		return Result{}, fmt.Errorf("index destination: %w", err)
	}
	fmt.Fprintf(opts.Out, "indexed %d existing files\n", ix.Count())

	opts.emit(ProgressEvent{Phase: "walking", Message: opts.RemoteRoot})
	fmt.Fprintf(opts.Out, "walking remote %s/ ...\n", opts.RemoteRoot)
	// Walk media + sidecars in one pass; partition below. Sidecars are
	// only used to queue companion deletes — they're never pulled.
	allEntries, err := cl.Walk(opts.RemoteRoot, extUnion(MediaExts, SidecarExts))
	if err != nil {
		return Result{}, fmt.Errorf("walk remote: %w", err)
	}
	var files []afc.File
	sidecarsByStem := make(map[string]string)
	for _, f := range allEntries {
		ext := strings.ToLower(filepath.Ext(f.Name))
		switch {
		case SidecarExts[ext]:
			sidecarsByStem[stemPath(f.Path)] = f.Path
		case MediaExts[ext]:
			files = append(files, f)
		}
	}
	res := Result{Total: len(files)}
	fmt.Fprintf(opts.Out, "remote media files: %d (+ %d sidecars)\n", res.Total, len(sidecarsByStem))

	// Pre-filter by destination dedup. Track consecutive matches for --until-found.
	type job struct {
		remote afc.File
	}
	jobs := make([]job, 0, len(files))
	// pendingDeletes collects remote paths whose content is verified at
	// destination — only safe to delete from the device when --delete-after
	// is set AND --confirm-delete is set. The CLI/GUI hard-gate that.
	var pendingDeletes []string
	pendingSeen := make(map[string]bool)
	deletionsEnabled := opts.DeleteAfter && !opts.DryRun

	// queueDelete adds a remote path to pendingDeletes once, plus any
	// sidecar that shares its stem (so we don't leave orphan .AAE files
	// littering DCIM after the media is gone). NOT thread-safe — caller
	// holds mu when invoked from a worker goroutine; pre-filter caller
	// is single-goroutine and needs no locking.
	queueDelete := func(remotePath string) {
		if pendingSeen[remotePath] {
			return
		}
		pendingSeen[remotePath] = true
		pendingDeletes = append(pendingDeletes, remotePath)
		if sidecar, ok := sidecarsByStem[stemPath(remotePath)]; ok && !pendingSeen[sidecar] {
			pendingSeen[sidecar] = true
			pendingDeletes = append(pendingDeletes, sidecar)
		}
	}
	consecutiveMatches := 0
	for _, f := range files {
		if ix.Has(f.Name, f.Size) {
			res.PreSkipped++
			if deletionsEnabled {
				queueDelete(f.Path)
			}
			if opts.UntilFound > 0 {
				consecutiveMatches++
				if consecutiveMatches >= opts.UntilFound {
					res.UntilFoundStop = true
					fmt.Fprintf(opts.Out, "--until-found %d reached; stopping scan early\n", opts.UntilFound)
					break
				}
			}
			continue
		}
		consecutiveMatches = 0
		jobs = append(jobs, job{remote: f})
	}
	fmt.Fprintf(opts.Out, "to pull: %d (pre-skipped: %d)\n", len(jobs), res.PreSkipped)
	opts.emit(ProgressEvent{
		Phase:      "planning",
		Total:      len(jobs),
		PreSkipped: res.PreSkipped,
	})

	if opts.DryRun {
		for _, j := range jobs {
			fmt.Fprintf(opts.Out, "  DRY pull %s (%d bytes)\n", j.remote.Path, j.remote.Size)
		}
		if opts.DeleteAfter {
			if len(pendingDeletes) == 0 {
				fmt.Fprintln(opts.Out, "  DRY delete: nothing yet verified at destination")
			} else {
				fmt.Fprintf(opts.Out, "  DRY delete: would remove %d files from device\n", len(pendingDeletes))
			}
		}
		return res, nil
	}

	// Don't early-return when len(jobs) == 0. If everything's already
	// pre-skipped, the worker pool loop is a no-op (the channel closes
	// immediately) and we fall straight through to the deletion phase.
	// Skipping the body here was the bug: pre-skipped files (already at
	// dest) couldn't be queued for delete on subsequent runs, so users
	// who ran a clean pull then re-ran with --delete-after saw nothing
	// happen.

	t0 := time.Now()
	tmpRoot, err := os.MkdirTemp(filepath.Dir(opts.OutputRoot), ".dumpsock-staging-")
	if err != nil {
		return res, fmt.Errorf("staging tmpdir: %w", err)
	}
	defer os.RemoveAll(tmpRoot)

	// Worker pool. Each job: pull, read EXIF, pick destination, move.
	// AFC is single-connection-bound; pulling on multiple goroutines from
	// the same Client is not safe per go-ios. We pull serially, but read
	// EXIF + move concurrently.
	type pulled struct {
		remote   afc.File
		stagedAt string
	}
	pulledCh := make(chan pulled, opts.Parallel*2)
	var pullErr error

	var mu sync.Mutex // protects res counter fields + ix
	go func() {
		defer close(pulledCh)
		for _, j := range jobs {
			if ctx.Err() != nil {
				pullErr = ctx.Err()
				return
			}
			staged := filepath.Join(tmpRoot, sanitizeFilename(j.remote.Name))
			if err := cl.PullTo(j.remote.Path, staged); err != nil {
				mu.Lock()
				res.Errors++
				mu.Unlock()
				fmt.Fprintf(opts.Err, "  ERROR pull %s: %v\n", j.remote.Path, err)
				continue
			}
			pulledCh <- pulled{remote: j.remote, stagedAt: staged}
		}
	}()

	var wg sync.WaitGroup
	sem := make(chan struct{}, opts.Parallel)

	processed := int32(0)
	for p := range pulledCh {
		wg.Add(1)
		sem <- struct{}{}
		go func(p pulled) {
			defer wg.Done()
			defer func() { <-sem }()

			date, hasDate, _ := exr.Date(ctx, p.stagedAt)

			// --since / --until filter
			if hasDate && shouldFilter(date, opts.Since, opts.Until) {
				mu.Lock()
				res.Filtered++
				mu.Unlock()
				_ = os.Remove(p.stagedAt)
				return
			}

			var dateDir string
			if hasDate {
				dateDir = filepath.Join(opts.OutputRoot, date.Format("2006-01-02"))
			} else {
				dateDir = filepath.Join(opts.OutputRoot, noDateFolder)
				mu.Lock()
				res.NoDate++
				mu.Unlock()
			}
			if err := os.MkdirAll(dateDir, 0o755); err != nil {
				mu.Lock()
				res.Errors++
				mu.Unlock()
				fmt.Fprintf(opts.Err, "  ERROR mkdir %s: %v\n", dateDir, err)
				_ = os.Remove(p.stagedAt)
				return
			}

			target, skip := dedup.PickPath(dateDir, p.remote.Name, p.remote.Size, nil)
			if skip {
				mu.Lock()
				res.PostSkipped++
				// (name, size) match exists at destination — same content
				// as the remote file, verified at index time. Safe to
				// queue for device-side delete (along with any sidecar).
				if deletionsEnabled {
					queueDelete(p.remote.Path)
				}
				mu.Unlock()
				_ = os.Remove(p.stagedAt)
				return
			}

			if err := os.Rename(p.stagedAt, target); err != nil {
				// Cross-device or other rename failure — fall back to copy.
				if cerr := copyFile(p.stagedAt, target); cerr != nil {
					mu.Lock()
					res.Errors++
					mu.Unlock()
					fmt.Fprintf(opts.Err, "  ERROR move %s -> %s: %v\n", p.stagedAt, target, cerr)
					_ = os.Remove(p.stagedAt)
					return
				}
				_ = os.Remove(p.stagedAt)
			}

			// Size verification before queueing for device-side delete:
			// if the local file's byte size doesn't match the remote
			// stat, something is wrong — never queue that path for
			// deletion (data-loss avoidance).
			localInfo, statErr := os.Stat(target)
			localOK := statErr == nil && localInfo.Size() == p.remote.Size
			if !localOK {
				mu.Lock()
				res.Errors++
				mu.Unlock()
				fmt.Fprintf(opts.Err, "  ERROR size mismatch after move %s: local=%d remote=%d (will NOT delete from device)\n",
					target, localInfo.Size(), p.remote.Size)
				return
			}

			if opts.SetMtime && hasDate {
				_ = os.Chtimes(target, date, date)
			}

			mu.Lock()
			if filepath.Base(target) != p.remote.Name {
				res.Suffixed++
			}
			res.Pulled++
			ix.Add(filepath.Base(target), p.remote.Size)
			if deletionsEnabled {
				queueDelete(p.remote.Path)
			}
			mu.Unlock()

			n := atomic.AddInt32(&processed, 1)
			mu.Lock()
			snap := ProgressEvent{
				Phase:       "pulling",
				Done:        int(n),
				Total:       len(jobs),
				Current:     p.remote.Name,
				Pulled:      res.Pulled,
				PreSkipped:  res.PreSkipped,
				PostSkipped: res.PostSkipped,
				Suffixed:    res.Suffixed,
				NoDate:      res.NoDate,
				Filtered:    res.Filtered,
				Errors:      res.Errors,
			}
			mu.Unlock()
			opts.emit(snap)
			if int(n)%25 == 0 || int(n) == len(jobs) {
				rate := float64(n) / time.Since(t0).Seconds()
				fmt.Fprintf(opts.Out, "  %d/%d  pulled=%d skipped=%d suffixed=%d nodate=%d errors=%d  (%.1f files/s)\n",
					n, len(jobs), res.Pulled, res.PostSkipped, res.Suffixed, res.NoDate, res.Errors, rate)
			}
		}(p)
	}
	wg.Wait()
	res.Elapsed = time.Since(t0)
	if pullErr != nil && !errors.Is(pullErr, context.Canceled) {
		return res, pullErr
	}

	if deletionsEnabled {
		runDeletions(ctx, cl, &opts, pendingDeletes, &res)
	} else if opts.DeleteAfter && opts.DryRun {
		fmt.Fprintf(opts.Out, "  --delete-after: would remove %d files from device (dry-run; nothing deleted)\n",
			len(pendingDeletes))
	}

	if opts.Notify {
		body := fmt.Sprintf("%d pulled, %d skipped, %d errors", res.Pulled, res.PreSkipped+res.PostSkipped, res.Errors)
		notify.Send("DumpSock — backup complete", body)
	}

	opts.emit(ProgressEvent{
		Phase:        "done",
		Done:         res.Pulled + res.PostSkipped + res.NoDate + res.Errors + res.Filtered,
		Total:        len(jobs),
		Pulled:       res.Pulled,
		PreSkipped:   res.PreSkipped,
		PostSkipped:  res.PostSkipped,
		Suffixed:     res.Suffixed,
		NoDate:       res.NoDate,
		Filtered:     res.Filtered,
		Errors:       res.Errors,
		Deleted:      res.Deleted,
		DeleteErrors: res.DeleteErrors,
	})

	return res, nil
}

// runDeletions iterates pendingDeletes and removes each remote file via
// AFC. Emits "deleting" progress events. Increments res.Deleted /
// res.DeleteErrors. Honors ctx cancellation between files.
func runDeletions(ctx context.Context, cl *afc.Client, opts *Options, paths []string, res *Result) {
	if len(paths) == 0 {
		fmt.Fprintln(opts.Out, "  --delete-after: nothing verified at destination — no device-side deletions")
		return
	}
	fmt.Fprintf(opts.Out, "  --delete-after: removing %d verified files from device…\n", len(paths))
	opts.emit(ProgressEvent{Phase: "deleting", Total: len(paths)})

	for i, rpath := range paths {
		if ctx.Err() != nil {
			fmt.Fprintf(opts.Err, "  --delete-after: cancelled at %d/%d\n", i, len(paths))
			break
		}
		if err := cl.Remove(rpath); err != nil {
			res.DeleteErrors++
			fmt.Fprintf(opts.Err, "  --delete-after: could not remove %s: %v\n", rpath, err)
		} else {
			res.Deleted++
		}
		if (i+1)%25 == 0 || i+1 == len(paths) {
			opts.emit(ProgressEvent{
				Phase:        "deleting",
				Done:         i + 1,
				Total:        len(paths),
				Current:      rpath,
				PreSkipped:   res.PreSkipped,
				PostSkipped:  res.PostSkipped,
				Pulled:       res.Pulled,
				Errors:       res.Errors,
			})
			fmt.Fprintf(opts.Out, "  --delete-after: %d/%d  deleted=%d errors=%d\n",
				i+1, len(paths), res.Deleted, res.DeleteErrors)
		}
	}
}

func shouldFilter(date, since, until time.Time) bool {
	if !since.IsZero() && date.Before(since) {
		return true
	}
	if !until.IsZero() && date.After(until.Add(24*time.Hour-time.Nanosecond)) {
		return true
	}
	return false
}

func defaultOutput(deviceName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	clean := strings.Map(func(r rune) rune {
		if r == os.PathSeparator || r == '/' || r == '\\' {
			return '_'
		}
		return r
	}, deviceName)
	if clean == "" {
		clean = "iPhone"
	}
	return filepath.Join(home, "DumpSock", clean), nil
}

func sanitizeFilename(name string) string {
	// AFC names are POSIX-clean; this is defense-in-depth for staging.
	name = filepath.Base(name)
	if name == "." || name == ".." || name == "" {
		return "_unnamed"
	}
	return name
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}
