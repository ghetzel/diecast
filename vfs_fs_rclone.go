package diecast


import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	_ "github.com/rclone/rclone/backend/all"
	rclone_fs "github.com/rclone/rclone/fs"
	rclone_config "github.com/rclone/rclone/fs/config"
	rclone_configfile "github.com/rclone/rclone/fs/config/configfile"
)


func init() {
	// RegisterFS(`s3`, func(layer *Layer) fs.FS {
	// 	var bucket = layer.Option(`bucket`).String()
	// })
}


