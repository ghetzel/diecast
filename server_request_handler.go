package diecast

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type candidateFile struct {
	Type          string
	Source        string
	Path          string
	Data          http.File
	StatusCode    int
	MimeType      string
	RedirectTo    string
	RedirectCode  int
	Headers       map[string]interface{}
	PathParams    []KV
	ForceTemplate bool
}

// The main entry point for handling requests not otherwise intercepted by Actions or User Routes.
//
// The Process:
//     1. Build a list of paths to try based on the requested path.  This is how things like
//        expanding "/thing" -> "/thing/index.html" OR "/thing.html" works.
//
//     2. For each path, do the following:
//
//        a. try to find a local file named X in the webroot
//        b.
//
func (self *Server) handleRequest(w http.ResponseWriter, req *http.Request) {
	var id = reqid(req)
	var prefix = fmt.Sprintf("%s/", self.rp())
	var lastErr error
	var serveFile *candidateFile

	if strings.HasPrefix(req.URL.Path, prefix) {
		defer req.Body.Close()

		// get a sequence of paths to search
		var requestPaths = self.candidatePathsForRequest(req)
		var localCandidate *candidateFile
		var mountCandidate *candidateFile
		var autoindexCandidate *candidateFile

		// SEARCH PHASE: locate a pile of data to serve and possibly templatize
		// -----------------------------------------------------------------------------------------

		// search for local files and autoindex opportunities
		for _, rPath := range requestPaths {
			// log.Debugf("[%s] try local file: %s", id, rPath)

			// actually try to stat the file from the filesystem rooted at RootPath
			if file, mimetype, err := self.tryLocalFile(rPath, req); err == nil {
				if localCandidate == nil {
					// log.Debugf("[%s] found local file: %s", id, rPath)
					localCandidate = &candidateFile{
						Type:     `local`,
						Source:   httpFilename(file),
						Path:     rPath,
						Data:     file,
						MimeType: mimetype,
					}
				} else {
					break
				}
			} else if IsDirectoryErr(err) {
				if archive, mimetype, err := self.streamAutoArchiveDirectory(file, rPath, req); err == nil {
					localCandidate = &candidateFile{
						Type:     `local`,
						Source:   httpFilename(archive),
						Path:     rPath,
						Data:     archive,
						MimeType: mimetype,
					}
				} else if self.Autoindex {
					if file, mimetype, ok := self.tryAutoindex(); ok {
						if autoindexCandidate == nil {
							// log.Debugf("[%s] found autoindex template for %s", id, rPath)
							autoindexCandidate = &candidateFile{
								Type:          `autoindex`,
								Source:        httpFilename(file),
								Path:          rPath,
								Data:          file,
								MimeType:      mimetype,
								ForceTemplate: true,
							}
						}
					}
				}
			}
		}

		// if we're not interested in local files first (therefore, we need to try the mounts anyway), or
		// we ARE trying local first, but nothing came back, then try searching mounts
		if !self.TryLocalFirst || localCandidate == nil {
			for _, rPath := range requestPaths {
				if mount, mountResponse, err := self.tryMounts(rPath, req); err == nil && mountResponse != nil {
					if mountCandidate == nil {
						// log.Debugf("[%s] found mount response: %s", id, rPath)
						mountCandidate = &candidateFile{
							Type:         `mount`,
							Source:       mountSummary(mount),
							Path:         rPath,
							Data:         mountResponse.GetFile(),
							MimeType:     mountResponse.ContentType,
							StatusCode:   mountResponse.StatusCode,
							Headers:      mountResponse.Metadata,
							RedirectTo:   mountResponse.RedirectTo,
							RedirectCode: mountResponse.RedirectCode,
						}
					}

					break

				} else if IsHardStop(err) {
					// A mount Hard Stop means:
					//  1. a mount handled the thing
					//  2. that mount explicitly wants us to return an error
					//  3. it doesn't matter if there are other candidate files, Hard Stop wins
					lastErr = err
					break
				}
			}
		}

		if localCandidate != nil {
			defer localCandidate.Data.Close()
		}

		if mountCandidate != nil {
			defer mountCandidate.Data.Close()
		}

		if autoindexCandidate != nil {
			defer autoindexCandidate.Data.Close()
		}

		if lastErr == nil {
			if self.TryLocalFirst {
				if localCandidate != nil {
					serveFile = localCandidate
				} else if mountCandidate != nil {
					serveFile = mountCandidate
				} else if autoindexCandidate != nil {
					serveFile = autoindexCandidate
				}
			} else {
				if mountCandidate != nil {
					serveFile = mountCandidate
				} else if localCandidate != nil {
					serveFile = localCandidate
				} else if autoindexCandidate != nil {
					serveFile = autoindexCandidate
				}
			}

			if serveFile != nil {
				log.Debugf("[%s] found: %s (%v)", id, serveFile.Type, serveFile.Source)

				if strings.Contains(serveFile.Path, `__id.`) {
					var value = strings.Trim(path.Base(req.URL.Path), `/`)

					serveFile.PathParams = append(serveFile.PathParams, KV{
						K: `id`,
						V: typeutil.Auto(value),
					})
				}

				if rcode := serveFile.RedirectCode; rcode > 0 {
					if serveFile.RedirectTo == `` {
						serveFile.RedirectTo = fmt.Sprintf("%s/", req.URL.Path)
					}

					http.Redirect(w, req, serveFile.RedirectTo, rcode)
					log.Debugf("[%s] path %v redirecting to %v (HTTP %d)", id, serveFile.Path, serveFile.RedirectTo, rcode)
					return
				} else if handled := self.handleCandidateFile(w, req, serveFile); handled {
					return
				}
			}
		}
	}

	if self.hasUserRoutes {
		self.userRouter.ServeHTTP(w, req)
	} else if lastErr != nil {
		// something else went sideways
		self.respondError(w, req, fmt.Errorf("an error occurred accessing %s: %v", req.URL.Path, lastErr), http.StatusServiceUnavailable)
	} else {
		// if we got *here*, then File Not Found
		self.respondError(w, req, fmt.Errorf("file %q was not found.", req.URL.Path), http.StatusNotFound)
	}
}

func (self *Server) candidatePathsForRequest(req *http.Request) []string {
	var requestPaths = []string{
		req.URL.Path,
	}

	// if we're looking at a directory, throw in the index file if the path as given doesn't respond
	if strings.HasSuffix(req.URL.Path, `/`) {
		requestPaths = append(requestPaths, path.Join(req.URL.Path, self.IndexFile))

		for _, ext := range self.TryExtensions {
			var base = filepath.Base(self.IndexFile)
			base = strings.TrimSuffix(base, filepath.Ext(self.IndexFile))

			requestPaths = append(requestPaths, path.Join(req.URL.Path, fmt.Sprintf("%s.%s", base, ext)))
		}

		for _, ext := range self.TryExtensions {
			requestPaths = append(requestPaths, strings.TrimSuffix(req.URL.Path, `/`)+`.`+ext)
		}

	} else if path.Ext(req.URL.Path) == `` {
		// if we're requesting a path without a file extension, try an index file in a directory with that name,
		// then try just <filename>.html
		requestPaths = append(requestPaths, fmt.Sprintf("%s/%s", req.URL.Path, self.IndexFile))

		for _, ext := range self.TryExtensions {
			requestPaths = append(requestPaths, fmt.Sprintf("%s.%s", req.URL.Path, ext))
		}
	}

	// finally, add handlers for implementing routing
	if parent := path.Dir(req.URL.Path); parent != `.` {
		for _, ext := range self.TryExtensions {
			requestPaths = append(requestPaths, fmt.Sprintf("%s/index__id.%s", strings.TrimSuffix(parent, `/`), ext))

			if base := strings.TrimSuffix(parent, `/`); base != `` {
				requestPaths = append(requestPaths, fmt.Sprintf("%s__id.%s", base, ext))
			}
		}
	}

	// unique the request paths
	requestPaths = sliceutil.UniqueStrings(requestPaths)

	// trim RoutePrefix from the front of all paths
	for i, rPath := range requestPaths {
		requestPaths[i] = strings.TrimPrefix(rPath, self.rp())
	}

	return requestPaths
}

func (self *Server) handleCandidateFile(
	w http.ResponseWriter,
	req *http.Request,
	file *candidateFile,
) bool {
	if file == nil {
		return false
	}

	// add in any metadata as response headers
	for k, v := range file.Headers {
		w.Header().Set(k, fmt.Sprintf("%v", v))
	}

	if file.MimeType == `` {
		file.MimeType = fileutil.GetMimeType(file.Path, `application/octet-stream`)
	}

	// set Content-Type
	w.Header().Set(`Content-Type`, file.MimeType)

	// write out the HTTP status if we were given one
	if file.StatusCode > 0 {
		w.WriteHeader(file.StatusCode)
	}

	// we got a real actual file here, figure out if we're templating it or not
	if file.ForceTemplate || self.shouldApplyTemplate(file.Path) {
		// tease the template header out of the file
		if header, templateData, err := SplitTemplateHeaderContent(file.Data); err == nil {
			// render the final template and write it out
			if err := self.applyTemplate(
				w,
				req,
				file.Path,
				templateData,
				header,
				file.PathParams,
				file.MimeType,
			); err != nil {
				self.respondError(w, req, fmt.Errorf("render template: %v", err), http.StatusInternalServerError)
			}
		} else {
			self.respondError(w, req, fmt.Errorf("parse template: %v", err), http.StatusInternalServerError)
		}
	} else {
		// if not templated, then the file is returned outright
		if rendererName := httputil.Q(req, `renderer`); rendererName == `` {
			io.Copy(w, file.Data)
		} else if renderer, err := GetRenderer(rendererName, self); err == nil {
			if err := renderer.Render(w, req, RenderOptions{
				Input: file.Data,
			}); err != nil {
				self.respondError(w, req, err, http.StatusInternalServerError)
			}
		} else if renderer, ok := GetRendererForFilename(file.Path, self); ok {
			if err := renderer.Render(w, req, RenderOptions{
				Input: file.Data,
			}); err != nil {
				self.respondError(w, req, err, http.StatusInternalServerError)
			}
		} else {
			self.respondError(w, req, fmt.Errorf("Unknown renderer %q", rendererName), http.StatusBadRequest)
		}
	}

	return true
}

func (self *Server) streamAutoArchiveDirectory(root http.File, requestPath string, req *http.Request) (http.File, string, error) {
	if !self.shouldAutocompress(requestPath) {
		return nil, ``, io.EOF
	}

	// start new zip archive
	var buf bytes.Buffer
	var archive = zip.NewWriter(&buf)

	// walk all files recursively under root
	if err := self.walkHttpFile(requestPath, root, func(path string, f http.File, s os.FileInfo) error {
		defer f.Close()

		path = strings.TrimPrefix(path, requestPath)
		path = strings.TrimPrefix(path, `/`)

		if path != `` {
			if file, err := archive.Create(path); err == nil {
				if _, err := io.Copy(file, f); err != nil {
					return err
				}
			} else {
				return err
			}
		}

		return nil
	}); err == nil {
		if err := archive.Close(); err != nil {
			return nil, ``, err
		}

		var filename = filepath.Base(requestPath)
		var outfile = newHttpFile(filename, buf.Bytes())

		return outfile, fileutil.GetMimeType(filename), nil
	} else {
		return nil, ``, err
	}
}

func (self *Server) walkHttpFile(path string, startFile http.File, fileFn func(string, http.File, os.FileInfo) error) error {
	if s, err := startFile.Stat(); err == nil {
		if s.IsDir() {
			path = strings.TrimSuffix(path, `/`) + `/`
		}

		if s.IsDir() {
			for {
				if children, err := startFile.Readdir(1024); err == nil {
					if len(children) == 0 {
						break
					}

					for _, c := range children {
						var subpath = filepath.Join(path, c.Name())

						if subfile, err := self.fs.Open(subpath); err == nil {
							var err = self.walkHttpFile(subpath, subfile, fileFn)

							subfile.Close()

							if err != nil {
								return err
							}
						}
					}
				} else {
					break
				}
			}
		} else if err := fileFn(path, startFile, s); err != nil {
			return err
		}
	}

	return nil
}

func httpFilename(file http.File) string {
	if stat, err := file.Stat(); err == nil {
		return stat.Name()
	}

	return `<unknown>`
}
