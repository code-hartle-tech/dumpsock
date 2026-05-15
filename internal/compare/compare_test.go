package compare

import (
	"testing"

	"github.com/code-hartle-tech/dumpsock/internal/afc"
)

func TestCategorize_LivePhotoPairing(t *testing.T) {
	items := []item{
		{name: "IMG_0001.HEIC", size: 2_500_000, ext: ".heic", stem: "img_0001"},
		{name: "IMG_0001.MOV", size: 3_500_000, ext: ".mov", stem: "img_0001"},
		{name: "IMG_0002.HEIC", size: 2_400_000, ext: ".heic", stem: "img_0002"},
		{name: "IMG_0003.MOV", size: 50_000_000, ext: ".mov", stem: "img_0003"},
		{name: "IMG_0004.PNG", size: 1_000_000, ext: ".png", stem: "img_0004"},
		{name: "Screenshot.png", size: 800_000, ext: ".png", stem: "screenshot"},
		{name: "doc.pdf", size: 200_000, ext: ".pdf", stem: "doc"},
		{name: "noise.bin", size: 100, ext: ".bin", stem: "noise"},
	}
	b := categorizeItems(items)

	// One Live Photo pair (IMG_0001), counted once with combined size.
	if b.LivePhotos.Items != 1 {
		t.Fatalf("live photos items = %d, want 1", b.LivePhotos.Items)
	}
	wantBytes := int64(2_500_000 + 3_500_000)
	if b.LivePhotos.Bytes != wantBytes {
		t.Errorf("live photos bytes = %d, want %d", b.LivePhotos.Bytes, wantBytes)
	}

	// One lone HEIC (IMG_0002) -> Photos.
	if b.Photos.Items != 1 || b.Photos.Bytes != 2_400_000 {
		t.Errorf("photos = %+v, want {1, 2400000}", b.Photos)
	}

	// One lone MOV (IMG_0003) -> Videos.
	if b.Videos.Items != 1 || b.Videos.Bytes != 50_000_000 {
		t.Errorf("videos = %+v, want {1, 50000000}", b.Videos)
	}

	// Two PNGs -> Screenshots (both, regardless of name prefix).
	if b.Screenshots.Items != 2 || b.Screenshots.Bytes != 1_800_000 {
		t.Errorf("screenshots = %+v, want {2, 1800000}", b.Screenshots)
	}

	// One PDF -> Documents.
	if b.Documents.Items != 1 {
		t.Errorf("documents items = %d, want 1", b.Documents.Items)
	}

	// One .bin -> Other.
	if b.Other.Items != 1 {
		t.Errorf("other items = %d, want 1", b.Other.Items)
	}
}

func TestBreakdown_Total(t *testing.T) {
	b := Breakdown{
		Photos:     Side{Items: 10, Bytes: 100},
		Videos:     Side{Items: 5, Bytes: 200},
		LivePhotos: Side{Items: 2, Bytes: 50},
	}
	total := b.Total()
	if total.Items != 17 || total.Bytes != 350 {
		t.Errorf("total = %+v, want {17, 350}", total)
	}
}

func TestDiffByNameSize(t *testing.T) {
	dev := mkAFC("IMG_0001.HEIC", 100, "IMG_0002.HEIC", 200, "IMG_0003.MOV", 300, "IMG_0004.HEIC", 400)
	back := []item{
		{name: "IMG_0001.HEIC", size: 100, stem: "img_0001", ext: ".heic"},
		{name: "IMG_0003.MOV", size: 999, stem: "img_0003", ext: ".mov"}, // same name, diff size
		{name: "IMG_0005.HEIC", size: 500, stem: "img_0005", ext: ".heic"},
	}
	newDev, newBackup, diff := diffByNameSize(dev, back)
	if newDev != 2 {
		t.Errorf("newDev = %d, want 2 (IMG_0002 + IMG_0004 only on device)", newDev)
	}
	if newBackup != 1 {
		t.Errorf("newBackup = %d, want 1 (IMG_0005 only at backup)", newBackup)
	}
	if diff != 1 {
		t.Errorf("diff = %d, want 1 (IMG_0003 differs)", diff)
	}
}

func mkAFC(args ...any) []afc.File {
	out := make([]afc.File, 0, len(args)/2)
	for i := 0; i+1 < len(args); i += 2 {
		out = append(out, afc.File{Name: args[i].(string), Size: int64(args[i+1].(int))})
	}
	return out
}
