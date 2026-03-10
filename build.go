package diecast

import (
	"encoding/xml"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/pkg/errors"
	"github.com/ryanuber/go-glob"
)

type BuildOptions struct {
	DestinationDir string   `json:"destdir"`
	IncludePaths   []string `json:"includes"`
	ExcludePaths   []string `json:"excludes"`
}

func (options *BuildOptions) prepareDestinationFile(path string, autosuffix bool) (string, error) {
	var dirname, filename = filepath.Split(
		strings.TrimPrefix(path, `/`),
	)

	dirname = filepath.Join(options.DestinationDir, dirname)
	filename = strings.TrimSuffix(filename, `/`)

	if filename == `` {
		filename = `index.html`
	} else if autosuffix && filepath.Ext(filename) == `` {
		filename += `.html`
	}

	var destfile = filepath.Join(dirname, filename)

	if !fileutil.DirExists(dirname) {
		if err := os.MkdirAll(dirname, 0755); err != nil {
			return ``, errors.Wrap(err, "mkdir")
		}
	}

	return destfile, nil
}

type Sitemap struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []SitemapURL `xml:"url"`
}

type SitemapURL struct {
	Location string `xml:"loc"`
}

func ParseSitemap(mapsource io.Reader) (*Sitemap, error) {
	var sitemap Sitemap

	if err := xml.NewDecoder(mapsource).Decode(&sitemap); err == nil {
		return &sitemap, nil
	} else {
		return nil, errors.Wrap(err, "parse")
	}
}

func BuildSite(server *Server, sitemap *Sitemap, options *BuildOptions) error {
	if options == nil {
		options = &BuildOptions{
			DestinationDir: `build`,
		}
	}

	for _, includePath := range options.IncludePaths {
		if err := fs.WalkDir(&server.VFS, includePath, func(path string, d fs.DirEntry, err error) error {
			for _, excludeGlob := range options.ExcludePaths {
				if glob.Glob(excludeGlob, path) {
					if d.IsDir() {
						return fs.SkipDir
					} else {
						return nil
					}
				} else if glob.Glob(excludeGlob, d.Name()) {
					if d.IsDir() {
						return fs.SkipDir
					} else {
						return nil
					}
				}
			}

			if err != nil {
				return err
			} else if !d.IsDir() {
				if destfile, err := options.prepareDestinationFile(path, false); err == nil {
					if source, err := server.VFS.Open(path); err == nil {
						defer source.Close()

						if file, err := os.Create(destfile); err == nil {
							var _, err = io.Copy(file, source)
							file.Close()

							if err != nil {
								return errors.Wrap(err, "write file "+destfile)
							}
						} else {
							return errors.Wrap(err, "open file "+destfile)
						}
					} else {
						return errors.Wrap(err, "read source "+path)
					}
				} else {
					return errors.Wrap(err, "prep file "+path)
				}
			}

			return nil
		}); err != nil {
			return errors.Wrap(err, "include path "+includePath)
		}
	}

	for _, loc := range sitemap.URLs {
		if url, err := url.Parse(loc.Location); err == nil {
			if res, err := server.SimulateRequest(``, url.Path, nil, nil, nil); err == nil {
				if res.Body != nil {
					defer res.Body.Close()

					if destfile, err := options.prepareDestinationFile(url.Path, true); err == nil {
						if file, err := os.Create(destfile); err == nil {
							var _, err = io.Copy(file, res.Body)
							file.Close()

							if err != nil {
								return errors.Wrap(err, "write file "+destfile)
							}
						} else {
							return errors.Wrap(err, "open file "+destfile)
						}
					} else {
						return errors.Wrap(err, "prep file "+url.Path)
					}
				}
			} else {
				log.Fatalf("request failed: %v", err)
			}
		} else {
			return errors.Wrap(err, "url "+loc.Location)
		}
	}

	return nil
}
