// Package frontend embeds the Wails GUI assets so the GUI binary stays a
// single self-contained file.
package frontend

import "embed"

//go:embed all:dist
var FS embed.FS
