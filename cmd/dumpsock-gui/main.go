// Command dumpsock-gui is the DumpSock Wails desktop application.
//
// DumpSock — by HARTLE.TECH · contact@hartle.tech
package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"github.com/code-hartle-tech/dumpsock/internal/frontend"
	"github.com/code-hartle-tech/dumpsock/internal/gui"
)

// assets is a stripped view of internal/frontend.FS rooted at "dist" so
// Wails serves files at the URL they're authored at (e.g. /index.html).
func mustSubFS() fs.FS {
	sub, err := fs.Sub(frontend.FS, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}

func main() {
	app := gui.NewApp()

	err := wails.Run(&options.App{
		Title:             "DumpSock",
		Width:             820,
		Height:            600,
		MinWidth:          640,
		MinHeight:         520,
		BackgroundColour:  &options.RGBA{R: 13, G: 13, B: 13, A: 255}, // #0D0D0D
		DisableResize:     false,
		Fullscreen:        false,
		HideWindowOnClose: false,
		OnStartup:         app.Startup,
		Bind:              []interface{}{app},
		AssetServer: &assetserver.Options{
			Assets: mustSubFS(),
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarHiddenInset(),
			Appearance:           mac.NSAppearanceNameDarkAqua,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   "DumpSock",
				Message: "Plug your phone in. Wring it dry.\n\nA HARTLE.TECH tool · contact@hartle.tech",
			},
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "dumpsock-gui: %s\n", err)
		os.Exit(1)
	}
}
