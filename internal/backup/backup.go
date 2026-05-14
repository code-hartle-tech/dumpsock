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

const noDateFolder = "0000:00:00 00:00:00"

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

	fmt.Fprintf(opts.Out, "indexing existing files under %s\n", opts.OutputRoot)
	ix, err := dedup.NewIndex(opts.OutputRoot)
	if err != nil {
		return Result{}, fmt.Errorf("index destination: %w", err)
	}
	fmt.Fprintf(opts.Out, "indexed %d existing files\n", ix.Count())

	fmt.Fprintf(opts.Out, "walking remote %s/ ...\n", opts.RemoteRoot)
	files, err := cl.Walk(opts.RemoteRoot, MediaExts)
	if err != nil {
		return Result{}, fmt.Errorf("walk remote: %w", err)
	}
	res := Result{Total: len(files)}
	fmt.Fprintf(opts.Out, "remote media files: %d\n", res.Total)

	// Pre-filter by destination dedup. Track consecutive matches for --until-found.
	type job struct {
		remote afc.File
	}
	jobs := make([]job, 0, len(files))
	consecutiveMatches := 0
	for _, f := range files {
		if ix.Has(f.Name, f.Size) {
			res.PreSkipped++
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

	if opts.DryRun {
		for _, j := range jobs {
			fmt.Fprintf(opts.Out, "  DRY %s (%d bytes)\n", j.remote.Path, j.remote.Size)
		}
		return res, nil
	}

	if len(jobs) == 0 {
		return res, nil
	}

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

			if opts.SetMtime && hasDate {
				_ = os.Chtimes(target, date, date)
			}

			mu.Lock()
			if filepath.Base(target) != p.remote.Name {
				res.Suffixed++
			}
			res.Pulled++
			ix.Add(filepath.Base(target), p.remote.Size)
			mu.Unlock()

			n := atomic.AddInt32(&processed, 1)
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

	if opts.DeleteAfter {
		fmt.Fprintln(opts.Err, "  --delete-after: device-side delete is not wired in v0; skipping")
	}

	if opts.Notify {
		body := fmt.Sprintf("%d pulled, %d skipped, %d errors", res.Pulled, res.PreSkipped+res.PostSkipped, res.Errors)
		notify.Send("DumpSock — backup complete", body)
	}

	return res, nil
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
