package diecast

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadStandardFunctionsPath(funcs FuncMap, server *Server) funcGroup {
	return funcGroup{
		Name:        `File Path Manipulation`,
		Description: `Used to parse and extract data from strings representing paths in a filesystem or tree hierarchy.`,
		Functions: []funcDef{
			{
				Name:    `basename`,
				Summary: `Return the filename component of the given path.`,
				Arguments: []funcArg{
					{
						Name:        `path`,
						Type:        `string`,
						Description: `The path to extract the filename from.`,
					}, {
						Name:        `extension`,
						Type:        `string`,
						Optional:    true,
						Description: `The path to extract the filename from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `basename "/this/is/my/file.jpg"`,
						Return: `file.jpg`,
					}, {
						Code:   `basename "/this/is/my/file.jpg" ".jpg"`,
						Return: `file`,
					},
				},
				Function: func(value interface{}, extnames ...string) string {
					var base = path.Base(fmt.Sprintf("%v", value))

					if ext := typeutil.String(sliceutil.FirstNonZero(extnames)); ext != `` {
						base = strings.TrimSuffix(base, ext)
					}

					return base
				},
			}, {
				Name:    `extname`,
				Summary: `Return the extension component of the given path (always prefixed with a dot [.]).`,
				Arguments: []funcArg{
					{
						Name:        `path`,
						Type:        `string`,
						Description: `The path to extract the file extension from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `extname "file.jpg"`,
						Return: `.jpg`,
					},
				},
				Function: func(value interface{}) string {
					return path.Ext(fmt.Sprintf("%v", value))
				},
			}, {
				Name:    `dirname`,
				Summary: `Return the directory path component of the given path.`,
				Arguments: []funcArg{
					{
						Name:        `path`,
						Type:        `string`,
						Description: `The path to extract the parent directory from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `dirname "/this/is/my/file.jpg"`,
						Return: `/this/is/my`,
					},
				},
				Function: func(value interface{}) string {
					return path.Dir(fmt.Sprintf("%v", value))
				},
			}, {
				Name:    `pathjoin`,
				Summary: `Return a string of all given path components joined together using the system path separator.`,
				Arguments: []funcArg{
					{
						Name:        `parts`,
						Type:        `strings`,
						Variadic:    true,
						Description: `One or more strings or string arrays to join together into a path.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `pathjoin "/this" "is/my" "file.jpg"`,
						Return: `/this/is/my/file.jpg`,
					},
				},
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
				Arguments: []funcArg{
					{
						Name:        `path`,
						Type:        `string`,
						Optional:    true,
						Description: `The path to retrieve an array of children from.`,
					},
				},
				Examples: []funcExample{
					{
						Code: `dir`,
						Return: []map[string]interface{}{
							{
								`name`:          `file.jpg`,
								`path`:          `/this/is/my/file.jpg`,
								`size`:          `124719`,
								`last_modified`: `2006-01-02T15:04:05Z07:00`,
								`directory`:     false,
								`mimetype`:      `image/jpeg`,
							}, {
								`name`:          `css`,
								`path`:          `/this/is/my/css`,
								`size`:          `4096`,
								`last_modified`: `2006-01-02T15:04:05Z07:00`,
								`directory`:     true,
							}, {
								`name`:          `README.md`,
								`path`:          `/this/is/my/README.md`,
								`size`:          `11216`,
								`last_modified`: `2006-01-02T15:04:05Z07:00`,
								`directory`:     false,
								`mimetype`:      `text/plain`,
							},
						},
					},
				},
				Function: func(dirs ...string) ([]map[string]interface{}, error) {
					var dir string
					var glob string
					var entries = make([]map[string]interface{}, 0)

					if len(dirs) == 0 || dirs[0] == `` || dirs[0] == `.` || dirs[0] == `/` {
						if server != nil {
							dir = `/`
						} else if wd, err := os.Getwd(); err == nil {
							dir = wd
						} else {
							return nil, err
						}
					} else {
						dir = dirs[0]
					}

					if len(dirs) > 1 {
						glob = dirs[1]
					}

					dir = path.Clean(dir)

					var ranAtLeastOnce bool

					if err := httpFsWalk(server.fs, dir, glob, func(path string, info os.FileInfo, err error) error {
						if err == nil {
							// omit first call (which is the root directory)
							if ranAtLeastOnce {
								var fi = &fileInfo{
									Parent:    filepath.Dir(path),
									Directory: info.IsDir(),
									FileInfo:  info,
								}

								entries = append(entries, fi.toMap())
							}
							// if matched, err := filepath.Match(dir, path); err == nil {
							// 	if matched {
							// 	}
							// } else {
							// 	return err
							// }

							if ranAtLeastOnce && info.IsDir() {
								// we only want one level, do not descend into directories
								return filepath.SkipDir
							} else {
								ranAtLeastOnce = true
								return nil
							}
						} else {
							return err
						}
					}); err == nil {
						sort.Slice(entries, func(i, j int) bool {
							var iname = typeutil.String(entries[i][`name`])
							var jname = typeutil.String(entries[j][`name`])

							return strings.ToLower(iname) < strings.ToLower(jname)
						})

						return entries, nil
					} else {
						return nil, err
					}
				},
			}, {
				Name:    `pathInRoot`,
				Summary: `Returns whether the given path falls within the Diecast serving root path.`,
				Arguments: []funcArg{
					{
						Name:        `path`,
						Type:        `string`,
						Description: `The path to check.`,
					},
				},
				Function: func(path string) bool {
					if server != nil {
						return server.IsInRootPath(path)
					} else {
						return true
					}
				},
			}, {
				Name:    `mimetype`,
				Summary: `Returns a best guess at the MIME type for the given filename.`,
				Arguments: []funcArg{
					{
						Name:        `filename`,
						Type:        `string`,
						Description: `The file to determine the type of.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `mimetype "file.jpg"`,
						Return: `image/jpeg`,
					}, {
						Code:   `mimetype "index.html"`,
						Return: `text/html`,
					},
				},
				Function: func(filename string) string {
					mime, _ := stringutil.SplitPair(fileutil.GetMimeType(path.Ext(filename)), `;`)
					return strings.TrimSpace(mime)
				},
			}, {
				Name:    `mimeparams`,
				Summary: `Returns the parameters portion of the MIME type of the given filename.`,
				Arguments: []funcArg{
					{
						Name:        `filename`,
						Type:        `string`,
						Description: `The file to retrieve MIME parameters from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `mimetype "file.jpg"`,
						Return: map[string]interface{}{},
					}, {
						Code: `mimetype "index.html"`,
						Return: map[string]interface{}{
							`charset`: `utf-8`,
						},
					},
				},
				Function: func(filename string) map[string]interface{} {
					_, params := stringutil.SplitPair(fileutil.GetMimeType(path.Ext(filename)), `;`)
					var kv = make(map[string]interface{})

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
