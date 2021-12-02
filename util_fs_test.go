package diecast

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/ghetzel/testify/require"
	"golang.org/x/tools/godoc/vfs/httpfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

func TestWalkHttpFileSystem(t *testing.T) {
	var testfs http.FileSystem = httpfs.New(mapfs.New(map[string]string{
		"zzz-last-file.txt":   "It should be visited last.",
		"a-file.txt":          "It has stuff.",
		"another-file.txt":    "Also stuff.",
		"folderA/entry-A.txt": "Alpha.",
		"folderA/entry-B.txt": "Beta.",
	}))

	var fileset = make([]string, 0)

	require.NoError(t, httpFsWalkFiles(testfs, `/`, func(path string, fi os.FileInfo, rs io.ReadSeeker, err error) error {
		require.NoError(t, err)

		if !fi.IsDir() {
			fileset = append(fileset, path)
		}

		return nil
	}))

	require.Equal(t, []string{
		`/a-file.txt`,
		`/another-file.txt`,
		`/folderA/entry-A.txt`,
		`/folderA/entry-B.txt`,
		`/zzz-last-file.txt`,
	}, fileset)
}
