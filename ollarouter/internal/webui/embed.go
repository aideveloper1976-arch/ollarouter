package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var EmbeddedFiles embed.FS

// EmbeddedFS returns the embedded filesystem rooted at "dist".
func EmbeddedFS() fs.FS {
	sub, err := fs.Sub(EmbeddedFiles, "dist")
	if err != nil {
		// Should never happen with valid embed
		panic(err)
	}
	return sub
}
