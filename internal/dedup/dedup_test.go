package dedup

import (
	"path/filepath"
	"testing"
)

// fakeStat returns canned (size, exists) for a synthetic destination tree.
func fakeStat(canned map[string]int64) func(string) (int64, bool) {
	return func(p string) (int64, bool) {
		if sz, ok := canned[p]; ok {
			return sz, true
		}
		return 0, false
	}
}

func TestPickPath(t *testing.T) {
	dir := "/dest/2026-05-14"
	tests := []struct {
		name      string
		basename  string
		size      int64
		canned    map[string]int64
		wantPath  string
		wantSkip  bool
	}{
		{
			name:     "no conflict — straight write",
			basename: "IMG_0001.HEIC",
			size:     12345,
			canned:   map[string]int64{},
			wantPath: filepath.Join(dir, "IMG_0001.HEIC"),
			wantSkip: false,
		},
		{
			name:     "exact (name, size) at primary — skip",
			basename: "IMG_0001.HEIC",
			size:     12345,
			canned:   map[string]int64{filepath.Join(dir, "IMG_0001.HEIC"): 12345},
			wantPath: filepath.Join(dir, "IMG_0001.HEIC"),
			wantSkip: true,
		},
		{
			name:     "same name, different size — try -1 suffix",
			basename: "IMG_0001.HEIC",
			size:     99999,
			canned:   map[string]int64{filepath.Join(dir, "IMG_0001.HEIC"): 12345},
			wantPath: filepath.Join(dir, "IMG_0001-1.HEIC"),
			wantSkip: false,
		},
		{
			name:     "primary collides + -1 has matching size — skip",
			basename: "IMG_0001.HEIC",
			size:     99999,
			canned: map[string]int64{
				filepath.Join(dir, "IMG_0001.HEIC"):   12345,
				filepath.Join(dir, "IMG_0001-1.HEIC"): 99999,
			},
			wantPath: filepath.Join(dir, "IMG_0001-1.HEIC"),
			wantSkip: true,
		},
		{
			name:     "primary, -1, -2 all taken with diff sizes — write -3",
			basename: "IMG_0001.HEIC",
			size:     77777,
			canned: map[string]int64{
				filepath.Join(dir, "IMG_0001.HEIC"):   12345,
				filepath.Join(dir, "IMG_0001-1.HEIC"): 22222,
				filepath.Join(dir, "IMG_0001-2.HEIC"): 33333,
			},
			wantPath: filepath.Join(dir, "IMG_0001-3.HEIC"),
			wantSkip: false,
		},
		{
			name:     "no extension — suffix still works",
			basename: "raw_dump",
			size:     1024,
			canned:   map[string]int64{filepath.Join(dir, "raw_dump"): 2048},
			wantPath: filepath.Join(dir, "raw_dump-1"),
			wantSkip: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotSkip := PickPath(dir, tc.basename, tc.size, fakeStat(tc.canned))
			if gotPath != tc.wantPath {
				t.Errorf("path: got %q, want %q", gotPath, tc.wantPath)
			}
			if gotSkip != tc.wantSkip {
				t.Errorf("skip: got %v, want %v", gotSkip, tc.wantSkip)
			}
		})
	}
}

func TestIndex_HasAdd(t *testing.T) {
	ix := &Index{bySize: make(map[string]map[int64]struct{})}
	if ix.Has("IMG.HEIC", 100) {
		t.Fatal("empty index should not have IMG.HEIC")
	}
	ix.Add("IMG.HEIC", 100)
	if !ix.Has("IMG.HEIC", 100) {
		t.Fatal("after Add(IMG.HEIC, 100), Has should be true")
	}
	if ix.Has("IMG.HEIC", 200) {
		t.Fatal("size 200 should not match a size 100 entry")
	}
	if got, want := ix.Count(), 1; got != want {
		t.Errorf("Count: got %d, want %d", got, want)
	}
}
