package diecast

import (
	"fmt"
	"github.com/ghetzel/diecast/v2/internal"
	rclone_fs "github.com/rclone/rclone/fs"
	"io/fs"
)

func init() {
	for _, reginfo := range rclone_fs.Registry {
		RegisterFS(reginfo.Prefix, func(layer *Layer) (fs.FS, error) {
			// fmt.Printf("vfs/%s: %v\n", layer.Type, layer.RootDir)
			return internal.CreateRcloneFilesystem(
				layer.String(),
				fmt.Sprintf("%s:%s", layer.Type, layer.RootDir),
				nil,
			)
		})
	}
}
