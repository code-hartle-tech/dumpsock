// Package notify fires native desktop notifications when a backup finishes.
// macOS uses osascript. Other platforms are no-op in v0 (Linux libnotify and
// Windows toast support are Phase 2 GUI work, not Phase 1 CLI).
package notify

import (
	"os/exec"
	"runtime"
	"strings"
)

// Send displays a system notification with the given title and body. Errors
// are intentionally swallowed — a missing notification is never worth
// failing a successful backup over.
func Send(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		sendDarwin(title, body)
	default:
		// no-op on linux/windows in v0
	}
}

func sendDarwin(title, body string) {
	script := `display notification "` + escape(body) + `" with title "` + escape(title) + `"`
	_ = exec.Command("osascript", "-e", script).Run()
}

func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
