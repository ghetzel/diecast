package diecast

import (
	"io/fs"
	"os"
)

var filesystems = make(map[string]FileSystemFunc)

func init() {
	// RegisterFS(`s3`, func(layer *Layer) fs.FS {
	// 	var bucket = layer.Option(`bucket`).String()
	// })

	RegisterFS(``, func(layer *Layer) (fs.FS, error) {
		var root = layer.RootDir

		if root == `` {
			root = `.`
		}

		return os.DirFS(root), nil
	})
}
