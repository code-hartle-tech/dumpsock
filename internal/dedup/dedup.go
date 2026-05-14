// Package dedup implements icloudpd's name-size-dedup-with-suffix policy.
//
//   - If a file with the same (basename, byte-size) already exists anywhere
//     under the destination tree, skip it.
//   - If the destination folder contains a file with the same basename but a
//     different size, store the new file with a `-1`, `-2`, ... suffix.
//   - Otherwise, save with the basename as-is.
//
// This package is pure logic; no I/O beyond the initial destination scan.
package dedup

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// Index maps basename -> set of byte sizes already present somewhere
// under the destination root. Built once at startup and (optionally) updated
// in-memory as new files land.
type Index struct {
	// name -> sizes (using map-as-set for O(1) lookup)
	bySize map[string]map[int64]struct{}
}

// NewIndex builds the destination index by walking root recursively.
// Missing root is not an error — it just yields an empty index.
func NewIndex(root string) (*Index, error) {
	ix := &Index{bySize: make(map[string]map[int64]struct{})}
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Missing root or unreadable subdir — skip rather than fail the whole scan.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		ix.Add(d.Name(), info.Size())
		return nil
	})
	return ix, walkErr
}

// Add records a (name, size) pair as present at destination.
func (ix *Index) Add(name string, size int64) {
	set, ok := ix.bySize[name]
	if !ok {
		set = make(map[int64]struct{})
		ix.bySize[name] = set
	}
	set[size] = struct{}{}
}

// Has reports whether the index already knows about (name, size).
func (ix *Index) Has(name string, size int64) bool {
	if set, ok := ix.bySize[name]; ok {
		_, exists := set[size]
		return exists
	}
	return false
}

// Count returns the total number of (name, size) records in the index.
func (ix *Index) Count() int {
	n := 0
	for _, s := range ix.bySize {
		n += len(s)
	}
	return n
}

// PickPath resolves the destination path for a file landing in dateDir with
// the given basename and byte size, honoring name+size dedup.
//
// Returns:
//
//	target     — absolute path to write to
//	skip       — true means a (name, size) match already exists at dateDir
//	             (or as a -N variant in the same dateDir); caller MUST NOT
//	             write a duplicate copy
//
// existsAt is a hook for testing — pass nil for production use; it falls
// back to os.Stat under the hood through filepath.Stat semantics.
func PickPath(dateDir, name string, size int64, statSize func(path string) (int64, bool)) (target string, skip bool) {
	if statSize == nil {
		statSize = defaultStatSize
	}
	primary := filepath.Join(dateDir, name)
	if existingSize, ok := statSize(primary); ok {
		if existingSize == size {
			return primary, true
		}
		// Same name, different size — try numbered variants.
		stem, ext := splitExt(name)
		for n := 1; ; n++ {
			candidate := filepath.Join(dateDir, joinSuffix(stem, n, ext))
			if cs, ok := statSize(candidate); ok {
				if cs == size {
					return candidate, true
				}
				continue
			}
			return candidate, false
		}
	}
	return primary, false
}

func splitExt(name string) (stem, ext string) {
	ext = filepath.Ext(name)
	stem = strings.TrimSuffix(name, ext)
	return
}

func joinSuffix(stem string, n int, ext string) string {
	// "IMG_0001.HEIC" + 2 -> "IMG_0001-2.HEIC"
	// Matches icloudpd's `name-size-dedup-with-suffix` numeric form.
	return stem + "-" + itoa(n) + ext
}

// itoa avoids importing strconv for a single-digit-likely path.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
