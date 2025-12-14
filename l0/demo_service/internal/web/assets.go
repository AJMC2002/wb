package web

import (
	"embed"
	"io/fs"
)

//go:embed *
var fsys embed.FS

func FS() (fs.FS, error) {
	return fs.Sub(fsys, ".")
}

func MustFS() fs.FS {
	f, err := FS()
	if err != nil {
		panic(err)
	}
	return f
}
