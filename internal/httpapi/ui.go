package httpapi

import (
	"embed"
	"io/fs"
)

//go:embed all:ui/dist
var embeddedUI embed.FS

func uiFS() fs.FS {
	sub, err := fs.Sub(embeddedUI, "ui/dist")
	if err != nil {
		panic(err)
	}

	return sub
}
