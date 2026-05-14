package dedup

import "os"

// defaultStatSize returns (size, true) if path exists as a regular file,
// (0, false) otherwise.
func defaultStatSize(path string) (int64, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0, false
	}
	return info.Size(), true
}
