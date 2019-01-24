package diecast

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/pathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsPath() funcGroup {
	return funcGroup{
		Name:        `File Path Manipulation`,
		Description: `Used to parse and extract data from strings representing paths in a filesystem or tree hierarchy.`,
		Functions: []funcDef{
			{
				Name:    `basename`,
				Summary: `Return the filename component of the given path.`,
				Function: func(value interface{}) string {
					return path.Base(fmt.Sprintf("%v", value))
				},
			}, {
				Name:    `extname`,
				Summary: `Return the extension component of the given path (always prefixed with a dot [.]).`,
				Function: func(value interface{}) string {
					return path.Ext(fmt.Sprintf("%v", value))
				},
			}, {
				Name:    `dirname`,
				Summary: `Return the directory path component of the given path.`,
				Function: func(value interface{}) string {
					return path.Dir(fmt.Sprintf("%v", value))
				},
			}, {
				Name:    `pathjoin`,
				Summary: `Return a string of all given path components joined together using the system path separator.`,
				Function: func(values ...interface{}) string {
					return path.Join(sliceutil.Stringify(sliceutil.Flatten(values))...)
				},
			}, {
				Name:     `pwd`,
				Summary:  `Return the present working directory.`,
				Function: os.Getwd,
			}, {
				Name:    `dir`,
				Summary: `Return a list of files and directories in *path*, or in the current directory if not specified.`,
				Function: func(dirs ...string) ([]*fileInfo, error) {
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

					dir = path.Clean(dir)

					if pathutil.DirExists(dir) {
						dir = path.Join(dir, `*`)
					}

					if e, err := filepath.Glob(dir); err == nil {
						for _, entry := range e {
							if info, err := os.Stat(entry); err == nil {
								entries = append(entries, &fileInfo{
									Parent:    path.Dir(entry),
									Directory: info.IsDir(),
									FileInfo:  info,
								})
							}
						}

						sort.Slice(entries, func(i, j int) bool {
							return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
						})

						return entries, nil
					} else {
						return nil, err
					}
				},
			}, {
				Name:    `mimetype`,
				Summary: `Returns a best guess at the MIME type for the given filename.`,
				Function: func(filename string) string {
					mime, _ := stringutil.SplitPair(fileutil.GetMimeType(path.Ext(filename)), `;`)
					return strings.TrimSpace(mime)
				},
			}, {
				Name:    `mimeparams`,
				Summary: `Returns the parameters portion of the MIME type of the given filename.`,
				Function: func(filename string) map[string]interface{} {
					_, params := stringutil.SplitPair(fileutil.GetMimeType(path.Ext(filename)), `;`)
					kv := make(map[string]interface{})

					for _, paramPair := range strings.Split(params, `;`) {
						key, value := stringutil.SplitPair(paramPair, `=`)
						kv[key] = stringutil.Autotype(value)
					}

					return kv
				},
			},
		},
	}
}
