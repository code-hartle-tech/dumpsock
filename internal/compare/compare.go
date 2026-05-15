// Package compare implements the "what's on the device vs what's on the
// destination" diff that powers the Compare & Merge tab in the GUI.
//
// The contract:
//
//   - Walk the device's DCIM via AFC for media + sidecars.
//   - Walk the destination folder tree recursively.
//   - Categorize every file into one of six buckets (see Category).
//   - Live-Photo pair detection: HEIC + MOV sharing a basename stem collapse
//     into a single Live Photo item.
//   - Compute three delta counts (new in device, new in backup, different
//     content under same name).
//
// Pure value-in / value-out. No goroutines, no IO beyond what the caller
// passed in. The GUI calls Compute() from a goroutine and pipes Progress
// callbacks back to JS.
package compare

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/code-hartle-tech/dumpsock/internal/afc"
)

// Category is the six-way classification rendered as a row in the Compare
// & Merge tab. The string values are the keys the frontend reads.
type Category string

const (
	CatPhotos      Category = "photos"
	CatVideos      Category = "videos"
	CatLivePhotos  Category = "live_photos"
	CatScreenshots Category = "screenshots"
	CatDocuments   Category = "documents"
	CatOther       Category = "other"
)

// AllCategories is the rendering order — matches the HTML markup.
var AllCategories = []Category{
	CatPhotos, CatVideos, CatLivePhotos, CatScreenshots, CatDocuments, CatOther,
}

// Side is a per-category count + byte total. Used twice in Result, once
// for the device and once for the backup.
type Side struct {
	Items int   `json:"items"`
	Bytes int64 `json:"bytes"`
}

// Breakdown is the per-side per-category split.
type Breakdown struct {
	Photos      Side `json:"photos"`
	Videos      Side `json:"videos"`
	LivePhotos  Side `json:"live_photos"`
	Screenshots Side `json:"screenshots"`
	Documents   Side `json:"documents"`
	Other       Side `json:"other"`
}

// Total returns the sum across categories.
func (b Breakdown) Total() Side {
	var s Side
	for _, side := range []Side{b.Photos, b.Videos, b.LivePhotos, b.Screenshots, b.Documents, b.Other} {
		s.Items += side.Items
		s.Bytes += side.Bytes
	}
	return s
}

// Add credits an item of size sz to category c.
func (b *Breakdown) Add(c Category, sz int64) {
	s := b.pick(c)
	s.Items++
	s.Bytes += sz
	b.assign(c, *s)
}

func (b *Breakdown) pick(c Category) *Side {
	switch c {
	case CatPhotos:
		return &b.Photos
	case CatVideos:
		return &b.Videos
	case CatLivePhotos:
		return &b.LivePhotos
	case CatScreenshots:
		return &b.Screenshots
	case CatDocuments:
		return &b.Documents
	default:
		return &b.Other
	}
}

func (b *Breakdown) assign(c Category, s Side) {
	switch c {
	case CatPhotos:
		b.Photos = s
	case CatVideos:
		b.Videos = s
	case CatLivePhotos:
		b.LivePhotos = s
	case CatScreenshots:
		b.Screenshots = s
	case CatDocuments:
		b.Documents = s
	default:
		b.Other = s
	}
}

// Result is the full Compare output shipped to JS.
type Result struct {
	Device      Breakdown     `json:"device"`
	Backup      Breakdown     `json:"backup"`
	NewInDevice int           `json:"new_in_device"`
	NewInBackup int           `json:"new_in_backup"`
	Different   int           `json:"different"`
	DeviceTotal Side          `json:"device_total"`
	BackupTotal Side          `json:"backup_total"`
	Elapsed     time.Duration `json:"elapsed_ns"`
}

// Progress is an in-flight status update. The GUI surfaces these as the
// "Walking…/Indexing…/Comparing…" line in the tab toolbar so the user knows
// it's not stuck.
type Progress struct {
	Phase   string `json:"phase"`
	Message string `json:"message,omitempty"`
}

// Options drives one Compute() invocation.
type Options struct {
	// Conn is an open AFC client positioned on the target device.
	Conn *afc.Client
	// RemoteRoot is the device-side directory to walk. Default "DCIM".
	RemoteRoot string
	// LocalRoot is the destination tree to walk. Must exist (we tolerate a
	// missing root as an empty backup).
	LocalRoot string
	// AllowedExts gates which file extensions count toward the breakdown.
	// Nil = use DefaultMediaExts. Extensions are case-insensitive, leading
	// dot ("." + "heic"). Sidecars (.aae) are ignored unconditionally.
	AllowedExts map[string]bool
	// OnProgress is invoked at phase boundaries. Optional.
	OnProgress func(Progress)
}

// DefaultMediaExts mirrors backup.MediaExts so a Compare run sees the same
// universe of files that a Backup run would pull.
var DefaultMediaExts = map[string]bool{
	".heic": true, ".heif": true,
	".jpg": true, ".jpeg": true, ".png": true,
	".mov": true, ".mp4": true, ".m4v": true,
	".dng": true, ".raw": true,
	".gif": true, ".webp": true,
	".pdf": true, ".txt": true, ".doc": true, ".docx": true,
}

// Compute runs the diff. Cancellation is honored between phases — a long
// walk can be interrupted by ctx.Done().
func Compute(ctx context.Context, opts Options) (Result, error) {
	if opts.Conn == nil {
		return Result{}, fmt.Errorf("compare: nil AFC client")
	}
	if opts.RemoteRoot == "" {
		opts.RemoteRoot = "DCIM"
	}
	exts := opts.AllowedExts
	if exts == nil {
		exts = DefaultMediaExts
	}
	emit := func(p Progress) {
		if opts.OnProgress != nil {
			opts.OnProgress(p)
		}
	}

	t0 := time.Now()
	emit(Progress{Phase: "walking_device", Message: opts.RemoteRoot})
	devFiles, err := opts.Conn.Walk(opts.RemoteRoot, exts)
	if err != nil {
		return Result{}, fmt.Errorf("compare: walk device: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	emit(Progress{Phase: "walking_backup", Message: opts.LocalRoot})
	backupFiles, err := walkLocal(opts.LocalRoot, exts)
	if err != nil {
		return Result{}, fmt.Errorf("compare: walk backup: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	emit(Progress{Phase: "categorizing"})
	dev := categorizeItems(from(devFiles))
	backup := categorizeItems(backupFiles)

	emit(Progress{Phase: "diffing"})
	newDev, newBackup, diff := diffByNameSize(devFiles, backupFiles)

	res := Result{
		Device:      dev,
		Backup:      backup,
		NewInDevice: newDev,
		NewInBackup: newBackup,
		Different:   diff,
		DeviceTotal: dev.Total(),
		BackupTotal: backup.Total(),
		Elapsed:     time.Since(t0),
	}
	emit(Progress{Phase: "done"})
	return res, nil
}

// item is the internal representation while computing.
type item struct {
	name string // basename
	size int64
	ext  string // lowercased with leading dot
	stem string // basename without extension, lowercased for pair matching
}

func walkLocal(root string, exts map[string]bool) ([]item, error) {
	var out []item
	if root == "" {
		return out, nil
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			// Missing root → empty backup. Don't fail the whole compare.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !exts[ext] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out = append(out, item{
			name: name,
			size: info.Size(),
			ext:  ext,
			stem: strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name))),
		})
		return nil
	})
	return out, err
}

// from converts an afc.File slice into the internal item form.
func from(afcFiles []afc.File) []item {
	out := make([]item, 0, len(afcFiles))
	for _, f := range afcFiles {
		ext := strings.ToLower(filepath.Ext(f.Name))
		out = append(out, item{
			name: f.Name,
			size: f.Size,
			ext:  ext,
			stem: strings.ToLower(strings.TrimSuffix(f.Name, filepath.Ext(f.Name))),
		})
	}
	return out
}

// categorize converts a list of afc.Files (device side) into a Breakdown.
func categorize(afcFiles []afc.File) Breakdown {
	return categorizeItems(from(afcFiles))
}

// categorizeItems is the inner form so we can reuse it for the local walk.
func categorizeItems(items []item) Breakdown {
	var b Breakdown

	// Live-photo pair detection: HEIC + MOV sharing a basename stem (case-
	// insensitive) collapse into one Live Photo entry. We do NOT also count
	// the HEIC as a Photo or the MOV as a Video — exactly one slot.
	heicStems := make(map[string]int64)
	movStems := make(map[string]int64)
	for _, it := range items {
		switch it.ext {
		case ".heic", ".heif":
			heicStems[it.stem] = it.size
		case ".mov":
			movStems[it.stem] = it.size
		}
	}
	livePairs := make(map[string]bool) // stems that are Live Photos
	for stem := range heicStems {
		if _, ok := movStems[stem]; ok {
			livePairs[stem] = true
		}
	}

	// Now classify every file. Skip the partner when its stem is a live pair.
	for _, it := range items {
		if livePairs[it.stem] {
			// Count the pair once on the HEIC; ignore the MOV partner.
			if it.ext == ".heic" || it.ext == ".heif" {
				heicSize := heicStems[it.stem]
				movSize := movStems[it.stem]
				b.Add(CatLivePhotos, heicSize+movSize)
			}
			continue
		}
		b.Add(classify(it), it.size)
	}
	return b
}

// classify is the per-file category heuristic. Live-Photo pairs are
// handled by the caller (categorizeItems) before this runs.
func classify(it item) Category {
	switch it.ext {
	case ".pdf", ".doc", ".docx", ".txt", ".rtf", ".xls", ".xlsx":
		return CatDocuments
	case ".mov", ".mp4", ".m4v":
		return CatVideos
	case ".png":
		// iOS screenshots are PNGs in DCIM. We don't have device-side
		// metadata that distinguishes "screenshot" from "art exported as
		// PNG"; on-device camera roll PNGs are 99% screenshots, so the
		// heuristic is "DCIM PNGs == screenshots". The 1% case lands in
		// Screenshots which is a tolerable miscategorization.
		if strings.HasPrefix(strings.ToLower(it.name), "screenshot") {
			return CatScreenshots
		}
		return CatScreenshots
	case ".heic", ".heif", ".jpg", ".jpeg", ".dng", ".raw", ".webp", ".gif":
		return CatPhotos
	default:
		return CatOther
	}
}

// diffByNameSize returns three counts:
//
//	newDev    — items present on the device but not at the backup
//	newBackup — items present at the backup but not on the device
//	diff      — items with the same basename present on both sides but
//	            with different sizes (likely an edit or re-encode)
//
// Matching is by basename, case-insensitive. Size mismatch counts toward
// "different"; size match counts as a normal match (not in any delta).
func diffByNameSize(device []afc.File, backup []item) (int, int, int) {
	type sizeSet struct{ sizes map[int64]struct{} }
	devByName := make(map[string]*sizeSet)
	for _, f := range device {
		k := strings.ToLower(f.Name)
		s, ok := devByName[k]
		if !ok {
			s = &sizeSet{sizes: map[int64]struct{}{}}
			devByName[k] = s
		}
		s.sizes[f.Size] = struct{}{}
	}
	backByName := make(map[string]*sizeSet)
	for _, it := range backup {
		k := strings.ToLower(it.name)
		s, ok := backByName[k]
		if !ok {
			s = &sizeSet{sizes: map[int64]struct{}{}}
			backByName[k] = s
		}
		s.sizes[it.size] = struct{}{}
	}

	var newDev, newBackup, different int

	for name, devSet := range devByName {
		backSet, ok := backByName[name]
		if !ok {
			newDev += len(devSet.sizes)
			continue
		}
		// Both sides know the name. Count "different" = sizes on device not
		// found in backup AND sizes in backup not found on device — minus
		// the matched ones (which are nothing).
		for sz := range devSet.sizes {
			if _, ok := backSet.sizes[sz]; !ok {
				different++
			}
		}
	}
	for name, backSet := range backByName {
		if _, ok := devByName[name]; !ok {
			newBackup += len(backSet.sizes)
		}
	}
	return newDev, newBackup, different
}
