package diecast

import (
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/pathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsPath(rv FuncMap) {
	// fn basename: Return the filename component of the given *path*.
	rv[`basename`] = func(value interface{}) string {
		return path.Base(fmt.Sprintf("%v", value))
	}

	// fn extname: Return the extension component of the given *path* (always prefixed with a dot [.]).
	rv[`extname`] = func(value interface{}) string {
		return path.Ext(fmt.Sprintf("%v", value))
	}

	// fn dirname: Return the directory path component of the given *path*.
	rv[`dirname`] = func(value interface{}) string {
		return path.Dir(fmt.Sprintf("%v", value))
	}

	// fn pathjoin: Return the value of all *values* join on the system path separator.
	rv[`pathjoin`] = func(values ...interface{}) string {
		return path.Join(sliceutil.Stringify(values)...)
	}

	// fn pwd: Return the present working directory
	rv[`pwd`] = os.Getwd

	// fn dir: Return a list of files and directories in *path*, or in the current directory if not specified.
	rv[`dir`] = func(dirs ...string) ([]*fileInfo, error) {
		var dir string
		entries := make([]*fileInfo, 0)

		if len(dirs) == 0 || dirs[0] == `` {
			if wd, err := os.Getwd(); err == nil {
				dir = wd
			} else {
				return nil, err
			}
		} else {
			dir = dirs[0]
		}

		if d, err := pathutil.ExpandUser(dir); err == nil {
			dir = d
		} else {
			return nil, err
		}

		if e, err := ioutil.ReadDir(dir); err == nil {
			for _, info := range e {
				entries = append(entries, &fileInfo{
					FileInfo: info,
				})
			}

			sort.Slice(entries, func(i, j int) bool {
				return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
			})

			return entries, nil
		} else {
			return nil, err
		}
	}

	// fn mimetype: Returns a best guess MIME type for the given filename
	rv[`mimetype`] = func(filename string) string {
		mime, _ := stringutil.SplitPair(mime.TypeByExtension(path.Ext(filename)), `;`)
		return strings.TrimSpace(mime)
	}

	// fn mimeparams: Returns the parameters portion of the MIME type of the given filename
	rv[`mimeparams`] = func(filename string) map[string]interface{} {
		_, params := stringutil.SplitPair(mime.TypeByExtension(path.Ext(filename)), `;`)
		rv := make(map[string]interface{})

		for _, paramPair := range strings.Split(params, `;`) {
			key, value := stringutil.SplitPair(paramPair, `=`)
			rv[key] = stringutil.Autotype(value)
		}

		return rv
	}
}
