package diecast

//go:generate esc -o static.go -pkg diecast -modtime 1500000000 -prefix ui ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/structs"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/pathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghodss/yaml"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"github.com/urfave/negroni"
)

var log = logging.MustGetLogger(`diecast`)

const DefaultAddress = `127.0.0.1:28419`
const DefaultRoutePrefix = `/`
const DefaultConfigFile = `diecast.yml`

var HeaderSeparator = []byte{'-', '-', '-'}
var DefaultIndexFile = `index.html`
var DefaultVerifyFile = `/` + DefaultIndexFile
var DefaultTemplatePatterns = []string{`*.html`}

type Server struct {
	Address             string           `json:"address"`
	Bindings            []Binding        `json:"bindings"`
	BindingPrefix       string           `json:"bindingPrefix"`
	RootPath            string           `json:"root"`
	LayoutPath          string           `json:"layouts"`
	ErrorsPath          string           `json:"errors"`
	EnableLayouts       bool             `json:"enableLayouts"`
	RoutePrefix         string           `json:"routePrefix"`
	TemplatePatterns    []string         `json:"patterns"`
	AdditionalFunctions template.FuncMap `json:"-"`
	TryLocalFirst       bool             `json:"localFirst"`
	IndexFile           string           `json:"indexFile"`
	VerifyFile          string           `json:"verifyFile"`
	Mounts              []Mount          `json:"-"`
	MountConfigs        []MountConfig    `json:"mounts"`
	BaseHeader          *TemplateHeader  `json:"header"`
	router              *httprouter.Router
	server              *negroni.Negroni
	fs                  http.FileSystem
	fsIsSet             bool
	fileServer          http.Handler
}

func NewServer(root string, patterns ...string) *Server {
	if len(patterns) == 0 {
		patterns = DefaultTemplatePatterns
	}

	return &Server{
		Address:          DefaultAddress,
		RoutePrefix:      DefaultRoutePrefix,
		RootPath:         root,
		EnableLayouts:    true,
		Bindings:         make([]Binding, 0),
		TemplatePatterns: patterns,
		IndexFile:        DefaultIndexFile,
		VerifyFile:       DefaultVerifyFile,
		Mounts:           make([]Mount, 0),
	}
}

func (self *Server) LoadConfig(filename string) error {
	if pathutil.FileExists(filename) {
		if file, err := os.Open(filename); err == nil {
			if data, err := ioutil.ReadAll(file); err == nil && len(data) > 0 {
				if err := yaml.Unmarshal(data, self); err == nil {
					// process mount configs into mount instances
					for i, config := range self.MountConfigs {
						if mount, err := NewMountFromSpec(fmt.Sprintf("%s:%s", config.Mount, config.To)); err == nil {
							mstruct := structs.New(mount)

							for k, v := range config.Options {
								for _, field := range mstruct.Fields() {
									if tag := field.Tag(`json`); tag != `` {
										if tag == k || strings.HasPrefix(tag, k+`,`) {
											if err := field.Set(v); err != nil {
												return fmt.Errorf("mount %d: field %v error: %v", i, k, err)
											}

											break
										}
									}
								}
							}

							self.Mounts = append(self.Mounts, mount)
						} else {
							return fmt.Errorf("invalid mount %d: %v", i, err)
						}
					}
				} else {
					return err
				}
			} else {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (self *Server) SetMounts(mounts []Mount) {
	if len(self.Mounts) > 0 {
		self.Mounts = append(self.Mounts, mounts...)
	} else {
		self.Mounts = mounts
	}
}

func (self *Server) SetFileSystem(fs http.FileSystem) {
	self.fs = fs
}

func (self *Server) Initialize() error {
	// always make sure the root path is absolute
	if v, err := filepath.Abs(self.RootPath); err == nil {
		cwd, err := os.Getwd()

		if v == `./` && err == nil {
			self.RootPath = cwd
		} else {
			self.RootPath = v
		}
	} else {
		return err
	}

	if self.LayoutPath == `` {
		self.LayoutPath = path.Join(`/`, `_layouts`)
	}

	if self.ErrorsPath == `` {
		self.ErrorsPath = path.Join(`/`, `_errors`)
	}

	self.RoutePrefix = strings.TrimSuffix(self.RoutePrefix, `/`)

	// if we haven't explicitly set a filesystem, create it
	if self.fs == nil {
		self.SetFileSystem(http.Dir(self.RootPath))
	}

	self.fileServer = http.FileServer(self.fs)

	if self.VerifyFile != `` {
		if _, err := self.fs.Open(self.VerifyFile); err != nil {
			return fmt.Errorf("Failed to open verification file %q: %v.", self.VerifyFile, err)
		}
	}

	if self.BindingPrefix != `` {
		log.Debugf("Binding prefix is %v", self.BindingPrefix)
	}

	for _, binding := range self.Bindings {
		binding.server = self
	}

	if err := self.setupServer(); err != nil {
		return err
	}

	return nil
}

func (self *Server) Serve() error {
	return http.ListenAndServe(self.Address, self.server)
}

func (self *Server) ListenAndServe(address string) error {
	self.Serve()
	return nil
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	self.server.ServeHTTP(w, req)
}

func (self *Server) shouldApplyTemplate(requestPath string) bool {
	baseName := filepath.Base(requestPath)

	for _, pattern := range self.TemplatePatterns {
		if strings.HasPrefix(pattern, `/`) {
			if match, err := filepath.Match(pattern, requestPath); err == nil && match {
				return true
			}
		} else {
			if match, err := filepath.Match(pattern, baseName); err == nil && match {
				return true
			}
		}
	}

	return false
}

func (self *Server) applyTemplate(w http.ResponseWriter, req *http.Request, requestPath string, reader io.Reader, header *TemplateHeader, urlParams map[string]interface{}, mimeType string) error {
	finalTemplate := bytes.NewBuffer(nil)
	hasLayout := false
	forceSkipLayout := false
	headerOffset := 0
	headers := make([]*TemplateHeader, 0)
	layouts := make([]string, 0)

	if header != nil {
		headers = append(headers, header)

		if header.lines > 0 {
			headerOffset = header.lines - 1
		}

		if header.Layout != `` {
			if header.Layout == `false` || header.Layout == `none` {
				forceSkipLayout = true
			} else {
				layouts = append([]string{header.Layout}, layouts...)
			}
		}
	}

	// add in includes first
	if err := self.InjectIncludes(finalTemplate, header); err != nil {
		return err
	}

	// only process layouts if we're supposed to
	if self.EnableLayouts && !forceSkipLayout {
		// files starting with "_" are partials and should not have layouts applied
		if !strings.HasPrefix(path.Base(requestPath), `_`) {
			// if no layouts were explicitly specified, and a layout named "default" exists, add it to the list
			if len(layouts) == 0 {
				if _, err := self.LoadLayout(`default`); err == nil {
					layouts = append(layouts, `default`)
				}
			}

			if len(layouts) > 0 {
				for _, layoutName := range layouts {
					if layoutFile, err := self.LoadLayout(layoutName); err == nil {
						if layoutHeader, layoutData, err := self.SplitTemplateHeaderContent(layoutFile); err == nil {
							if layoutHeader != nil {
								headers = append([]*TemplateHeader{layoutHeader}, headers...)

								// add in layout includes
								if err := self.InjectIncludes(finalTemplate, layoutHeader); err != nil {
									return err
								}
							}

							finalTemplate.WriteString("{{/* BEGIN LAYOUT '" + layoutName + "' */}}\n")
							finalTemplate.WriteString("\n{{ define \"layout\" }}\n")
							finalTemplate.Write(layoutData)
							finalTemplate.WriteString("\n{{ end }}\n")
							finalTemplate.WriteString("{{/* BEGIN LAYOUT '" + layoutName + "' */}}\n")
						} else {
							return err
						}

						hasLayout = true
					} else {
						// we don't care if the default layout is missing
						if layoutName != `default` {
							return err
						}
					}
				}
			}
		}
	}

	if hasLayout {
		finalTemplate.WriteString("\n{{ define \"content\" }}\n")
	}

	if _, err := io.Copy(finalTemplate, reader); err != nil {
		return err
	}

	if hasLayout {
		finalTemplate.WriteString("\n{{ end }}\n")
	}

	var baseHeader TemplateHeader

	if self.BaseHeader != nil {
		baseHeader = *self.BaseHeader
	}

	finalHeader := &baseHeader

	for _, templateHeader := range headers {
		if fh, err := finalHeader.Merge(templateHeader); err == nil {
			finalHeader = fh
		} else {
			return err
		}
	}

	if finalHeader != nil {
		// and put any url route params in there too
		finalHeader.UrlParams = urlParams
	}

	if funcs, data, err := self.GetTemplateData(req, finalHeader); err == nil {
		// create the template and make it aware of our custom functions
		tmpl := NewTemplate(
			self.ToTemplateName(requestPath),
			GetEngineForFile(requestPath),
		)

		tmpl.Funcs(funcs)
		tmpl.SetHeaderOffset(headerOffset)
		tmpl.SetPostProcessors(finalHeader.Postprocessors)

		if err := tmpl.Parse(finalTemplate.String()); err == nil {
			log.Debugf("Rendering %q as %v template (header offset by %d lines)", requestPath, tmpl.Engine(), headerOffset)

			if finalHeader != nil {
				// include any configured response headers now
				for name, value := range finalHeader.Headers {
					w.Header().Set(name, fmt.Sprintf("%v", value))
				}
			}

			if httputil.QBool(req, `__viewsource`) {
				w.Header().Set(`Content-Type`, `text/plain`)
				w.Write(finalTemplate.Bytes())
				return nil
			} else {
				w.Header().Set(`Content-Type`, mimeType)

				if hasLayout {
					return tmpl.Render(w, data, `layout`)
				} else {
					return tmpl.Render(w, data, ``)
				}
			}
		} else if httputil.QBool(req, `__viewsource`) {
			var tplstr string
			lines := strings.Split(finalTemplate.String(), "\n")
			lineNoSpaces := fmt.Sprintf("%d", len(fmt.Sprintf("%d", len(lines)))+1)

			for i, line := range lines {
				tplstr += fmt.Sprintf("% "+lineNoSpaces+"d | %s\n", i+1, line)
			}

			tplstr = fmt.Sprintf("ERROR: %v\n\n", err) + tplstr

			w.Header().Set(`Content-Type`, `text/plain`)
			w.Write([]byte(tplstr))
			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}

func (self *Server) GetTemplateFunctions(data interface{}) FuncMap {
	funcs := make(FuncMap)

	for k, v := range GetStandardFunctions() {
		funcs[k] = v
	}

	if self.AdditionalFunctions != nil {
		for k, v := range self.AdditionalFunctions {
			funcs[k] = v
		}
	}

	// fn payload: Return the body supplied with the request used to generate the current view.
	funcs[`payload`] = func(key ...string) interface{} {
		if len(key) == 0 {
			return data
		} else {
			return maputil.DeepGet(data, strings.Split(key[0], `.`), nil)
		}
	}

	// fn querystrings: Return a map of all of the query string parameters in the current URL.
	funcs[`querystrings`] = func() map[string]interface{} {
		if v := maputil.DeepGet(data, []string{`request`, `url`, `query`}, nil); v != nil {
			if vMap, ok := v.(map[string]interface{}); ok {
				return vMap
			}
		}

		return make(map[string]interface{})
	}

	// fn qs: Return the value of query string parameter *key* in the current URL, or return *fallback*.
	funcs[`qs`] = func(key interface{}, fallbacks ...interface{}) interface{} {
		if len(fallbacks) == 0 {
			fallbacks = []interface{}{nil}
		}

		return maputil.DeepGet(data, []string{`request`, `url`, `query`, fmt.Sprintf("%v", key)}, fallbacks[0])
	}

	// fn headers: Return the value of the *header* HTTP request header from the request used to
	//             generate the current view.
	funcs[`headers`] = func(key string) string {
		return fmt.Sprintf("%v", maputil.DeepGet(data, []string{`request`, `headers`, key}, ``))
	}

	// fn param: Return the value of the named or indexed URL parameter, or nil of none are present.
	funcs[`param`] = func(nameOrIndex interface{}) interface{} {
		if v := maputil.DeepGet(data, []string{
			`request`, `url`, `params`, fmt.Sprintf("%v", nameOrIndex),
		}, nil); v != nil {
			return stringutil.Autotype(v)
		} else {
			return nil
		}
	}

	// fn var: Set the runtime variable *name* to *value*.
	funcs[`var`] = func(name string, vI ...interface{}) interface{} {
		var value interface{}

		switch len(vI) {
		case 0:
			break
		case 1:
			value = vI[0]
		default:
			value = vI
		}

		maputil.DeepSet(data, []string{`vars`, name}, value)
		return ``
	}

	// fn push: Append to variable *name* to *value*.
	funcs[`push`] = func(name string, vI ...interface{}) interface{} {
		var values []interface{}

		if existing := maputil.DeepGet(data, []string{`vars`, name}); existing != nil {
			values = append(values, sliceutil.Sliceify(existing)...)
		}

		values = append(values, vI...)
		maputil.DeepSet(data, []string{`vars`, name}, values)

		return ``
	}

	// fn pop: Remove the last item from *name* and return it.
	funcs[`pop`] = func(name string) interface{} {
		var out interface{}

		if existing := maputil.DeepGet(data, []string{`vars`, name}); existing != nil {
			values := sliceutil.Sliceify(existing)

			switch len(values) {
			case 0:
				return nil
			case 1:
				out = values[0]
				maputil.DeepSet(data, []string{`vars`, name}, nil)
			default:
				out = values[len(values)-1]
				values = values[0 : len(values)-1]
				maputil.DeepSet(data, []string{`vars`, name}, values)
			}
		}

		return out
	}

	return funcs
}

func (self *Server) LoadLayout(name string) (io.Reader, error) {
	return self.fs.Open(fmt.Sprintf("%s/%s.html", self.LayoutPath, name))
}

func (self *Server) ToTemplateName(requestPath string) string {
	requestPath = strings.Replace(requestPath, `/`, `-`, -1)

	return requestPath
}

func (self *Server) GetTemplateData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]interface{}, error) {
	data := requestToEvalData(req, header)

	data[`vars`] = map[string]interface{}{}

	data[`diecast`] = map[string]interface{}{
		`binding_prefix`:    self.BindingPrefix,
		`route_prefix`:      self.RoutePrefix,
		`template_patterns`: self.TemplatePatterns,
		`try_local_first`:   self.TryLocalFirst,
		`index_file`:        self.IndexFile,
		`verify_file`:       self.VerifyFile,
	}

	bindings := make(map[string]interface{})
	bindingsToEval := make([]Binding, 0)

	// these are the functions that will be available to every part of the rendering process
	funcs := self.GetTemplateFunctions(data)

	bindingsToEval = append(bindingsToEval, self.Bindings...)

	if header != nil {
		bindingsToEval = append(bindingsToEval, header.Bindings...)
	}

	for _, binding := range bindingsToEval {
		binding.server = self

		if binding.Repeat == `` {
			bindings[binding.Name] = binding.Fallback
			data[`bindings`] = bindings

			if v, err := binding.Evaluate(req, header, data, funcs); err == nil && v != nil {
				bindings[binding.Name] = v
				data[`bindings`] = bindings
			} else {
				log.Warningf("Binding %q failed: %v", binding.Name, err)

				if !binding.Optional {
					return funcs, nil, err
				}
			}
		} else {
			results := make([]interface{}, 0)

			repeatExpr := fmt.Sprintf("{{ range $index, $item := (%v) }}\n", binding.Repeat)
			repeatExpr += fmt.Sprintf("%v\n", binding.Resource)
			repeatExpr += "{{ end }}"
			repeatExprOut := rxEmptyLine.ReplaceAllString(
				strings.TrimSpace(
					EvalInline(repeatExpr, data, funcs),
				),
				``,
			)

			log.Debugf("Repeater: \n%v\nOutput:\n%v", repeatExpr, repeatExprOut)
			repeatIters := strings.Split(repeatExprOut, "\n")

			for i, resource := range repeatIters {
				binding.Resource = strings.TrimSpace(resource)
				binding.Repeat = ``
				bindings[binding.Name] = binding.Fallback

				if v, err := binding.Evaluate(req, header, data, funcs); err == nil {
					results = append(results, v)
					bindings[binding.Name] = results
					data[`bindings`] = bindings
				} else {
					log.Warningf("Binding %q (iteration %d) failed: %v", binding.Name, i, err)

					if binding.OnError == ActionContinue {
						continue
					} else if binding.OnError == ActionBreak {
						break
					} else if !binding.Optional {
						return funcs, nil, err
					}
				}

				data[`bindings`] = bindings
			}
		}
	}

	data[`bindings`] = bindings

	if header != nil {
		flags := make(map[string]bool)

		for name, def := range header.FlagDefs {
			switch def.(type) {
			case bool:
				flags[name] = def.(bool)
				continue

			default:
				if v, err := stringutil.ConvertToBool(
					EvalInline(fmt.Sprintf("%v", def), data, funcs),
				); err == nil {
					flags[name] = v
					continue
				}
			}

			flags[name] = false
		}

		data[`flags`] = flags

		pageData := make(map[string]interface{})

		maputil.Walk(header.Page, func(value interface{}, path []string, isLeaf bool) error {

			if isLeaf {
				switch value.(type) {
				case string:
					value = EvalInline(value.(string), data, funcs)
					value = stringutil.Autotype(value)
				}

				maputil.DeepSet(pageData, path, value)
			}

			return nil
		})

		data[`page`] = pageData
	}

	return funcs, data, nil
}

func (self *Server) handleFileRequest(w http.ResponseWriter, req *http.Request) {
	log.Infof("%v %v", req.Method, req.URL)

	// normalize filename from request path
	requestPath := req.URL.Path

	requestPaths := []string{
		requestPath,
	}

	// if we're looking at a directory, throw in the index file if the path as given doesn't respond
	if strings.HasSuffix(requestPath, `/`) {
		requestPaths = append(requestPaths, path.Join(requestPath, self.IndexFile))
	} else if path.Ext(requestPath) == `` {
		// if we're requesting a path without a file extension, try an index file in a directory with that name,
		// then try just <filename>.html

		requestPaths = append(requestPaths, fmt.Sprintf("%s/%s", requestPath, self.IndexFile))
		requestPaths = append(requestPaths, fmt.Sprintf("%s.html", requestPath))
	}

	// finally, add handlers for implementing a junky form of url routing
	if parent := path.Dir(requestPath); parent != `.` {
		requestPaths = append(requestPaths, fmt.Sprintf("%s/index__id.html", parent))
		requestPaths = append(requestPaths, fmt.Sprintf("%s__id.html", parent))
	}

	var triedLocal bool

PathLoop:
	for _, rPath := range requestPaths {
		// remove the Route Prefix, as that's a structural part of the path but does not
		// represent where the files are (used for embedding diecast in other services
		// to avoid name collisions)
		//
		rPath = strings.TrimPrefix(rPath, self.RoutePrefix)
		var file http.File
		var mimeType string
		var message string
		var redirectTo string
		var redirectCode int
		var headers = make(map[string]interface{})
		var urlParams = make(map[string]interface{})

		log.Debugf("> trying path: %v", rPath)

		if self.TryLocalFirst && !triedLocal {
			triedLocal = true

			if f, m, err := self.tryLocalFile(rPath, req); err == nil {
				file = f
				mimeType = m
				message = fmt.Sprintf("< handled by filesystem")

			} else if mnt, response, err := self.tryMounts(rPath, req); err == nil {
				file = response.GetFile()
				mimeType = response.ContentType
				headers = response.Metadata
				redirectTo = response.RedirectTo
				redirectCode = response.RedirectCode
				message = fmt.Sprintf("< handled by %v after trying local first", mnt)

			} else if IsHardStop(err) {
				break PathLoop
			}
		} else {
			if mnt, response, err := self.tryMounts(rPath, req); err == nil && response != nil {
				file = response.GetFile()
				mimeType = response.ContentType
				headers = response.Metadata
				redirectTo = response.RedirectTo
				redirectCode = response.RedirectCode
				message = fmt.Sprintf("< handled by %v", mnt)

			} else if IsHardStop(err) {
				break PathLoop

			} else if f, m, err := self.tryLocalFile(rPath, req); err == nil {
				file = f
				mimeType = m
				message = fmt.Sprintf("< handled by filesystem")
			}
		}

		if redirectCode > 0 {
			if redirectTo == `` {
				redirectTo = fmt.Sprintf("%s/", req.URL.Path)
			}

			http.Redirect(w, req, redirectTo, redirectCode)
			log.Debugf("  path %v redirecting to %v (HTTP %d)", rPath, redirectTo, redirectCode)
			return
		}

		if file != nil {
			if strings.Contains(rPath, `__id.`) {
				urlParams[`1`] = strings.Trim(path.Base(req.URL.Path), `/`)
				urlParams[`id`] = strings.Trim(path.Base(req.URL.Path), `/`)
			}

			if handled := self.respondToFile(rPath, mimeType, file, headers, urlParams, w, req); handled {
				log.Debug(message)
				return
			}
		} else {
			log.Debugf("No mounts or filesystems handled path: %v", rPath)
		}
	}

	// if we got *here*, then File Not Found
	log.Debugf("< not found")

	self.respondError(w, fmt.Errorf("File %q was not found.", requestPath), http.StatusNotFound)
}

// Attempt to resolve the given path into a real file and return that file and mime type.
// Non-existent files, unreadable files, and directories will return an error.
func (self *Server) tryLocalFile(requestPath string, req *http.Request) (http.File, string, error) {
	// if we got here, try to serve the file from the filesystem
	if file, err := self.fs.Open(requestPath); err == nil {
		if stat, err := file.Stat(); err == nil {
			if !stat.IsDir() {
				mimeType := mime.TypeByExtension(path.Ext(stat.Name()))
				return file, mimeType, nil
			} else {
				return nil, ``, fmt.Errorf("is a directory")
			}
		} else {
			return nil, ``, fmt.Errorf("failed to stat file %v: %v", requestPath, err)
		}
	} else {
		return nil, ``, err
	}
}

func (self *Server) tryMounts(requestPath string, req *http.Request) (Mount, *MountResponse, error) {
	var body *bytes.Reader

	// buffer the request body because we need to repeatedly pass it to multiple mounts
	if data, err := ioutil.ReadAll(req.Body); err == nil {
		if len(data) > 0 {
			log.Debugf("  read %d bytes from request body\n%v", len(data), string(data))
		}

		body = bytes.NewReader(data)
	} else {
		return nil, nil, err
	}

	// find a mount that has this file
	for i, mount := range self.Mounts {
		log.Debugf("  trying mount %d %v", i, mount)

		// seek the body buffer back to the beginning
		if _, err := body.Seek(0, 0); err != nil {
			return nil, nil, err
		}

		if mount.WillRespondTo(requestPath, req, body) {
			// attempt to open the file entry
			if mountResponse, err := mount.OpenWithType(requestPath, req, body); err == nil {
				return mount, mountResponse, nil
			} else if IsHardStop(err) {
				return nil, nil, err
			} else {
				log.Warning(err)
			}
		}
	}

	return nil, nil, fmt.Errorf("%q not found", requestPath)
}

func (self *Server) respondToFile(requestPath string, mimeType string, file http.File, headers map[string]interface{}, urlParams map[string]interface{}, w http.ResponseWriter, req *http.Request) bool {
	// add in any metadata as response headers
	for k, v := range headers {
		w.Header().Set(k, fmt.Sprintf("%v", v))
	}

	if mimeType == `` {
		mimeType = `application/octet-stream`
	}

	// we got a real actual file here, figure out if we're templating it or not
	if self.shouldApplyTemplate(requestPath) {
		// tease the template header out of the file
		if header, templateData, err := self.SplitTemplateHeaderContent(file); err == nil {
			if header != nil {
				if redirect := header.Redirect; redirect != nil {
					w.Header().Set(`Location`, redirect.URL)

					if redirect.Code > 0 {
						w.WriteHeader(redirect.Code)
					} else {
						w.WriteHeader(http.StatusMovedPermanently)
					}

					return true
				}
			}

			// render the final template and write it out
			if err := self.applyTemplate(w, req, requestPath, bytes.NewBuffer(templateData), header, urlParams, mimeType); err != nil {
				self.respondError(w, err, http.StatusInternalServerError)
			}
		} else {
			self.respondError(w, err, http.StatusInternalServerError)
		}
	} else {
		w.Header().Set(`Content-Type`, mimeType)
		io.Copy(w, file)
	}

	return true
}

func (self *Server) respondError(w http.ResponseWriter, resErr error, code int) {
	tmpl := NewTemplate(`error`, HtmlEngine)

	if code >= 400 && code < 500 {
		log.Warningf("ERR %v (HTTP %d)", resErr, code)
	} else {
		log.Errorf("ERR %v (HTTP %d)", resErr, code)
	}

	if resErr == nil {
		resErr = fmt.Errorf("Unknown Error")
	}

	for _, filename := range []string{
		fmt.Sprintf("%s/%d.html", self.ErrorsPath, code),
		fmt.Sprintf("%s/%dxx.html", self.ErrorsPath, int(code/100.0)),
		fmt.Sprintf("%s/default.html", self.ErrorsPath),
	} {
		log.Debugf("> error path: %v", filename)

		if f, err := self.fs.Open(filename); err == nil {
			if err := tmpl.ParseFrom(f); err == nil {
				w.Header().Set(`Content-Type`, `text/html`)

				if err := tmpl.Render(w, map[string]interface{}{
					`error`: resErr.Error(),
				}, ``); err == nil {
					return
				} else {
					log.Warningf("Error template %v render failed: %v", filename, err)
				}
			} else {
				log.Warningf("Error template %v failed: %v", filename, err)
			}
		}
	}

	http.Error(w, resErr.Error(), code)
}

func (self *Server) SplitTemplateHeaderContent(reader io.Reader) (*TemplateHeader, []byte, error) {
	if data, err := ioutil.ReadAll(reader); err == nil {
		if bytes.HasPrefix(data, HeaderSeparator) {
			parts := bytes.SplitN(data, HeaderSeparator, 3)

			if len(parts) == 3 {
				header := TemplateHeader{}

				if parts[1] != nil {
					header.lines = len(strings.Split(string(parts[1]), "\n"))

					if err := yaml.Unmarshal(parts[1], &header); err != nil {
						return nil, nil, err
					}
				}

				return &header, parts[2], nil
			}
		}

		return nil, data, nil
	} else {
		return nil, nil, err
	}
}

func (self *Server) InjectIncludes(w io.Writer, header *TemplateHeader) error {
	includes := make(map[string]string)

	if header != nil {
		for name, includePath := range header.Includes {
			includes[name] = includePath
		}
	}

	if len(includes) > 0 {
		for name, includePath := range includes {
			if file, err := self.fs.Open(includePath); err == nil {
				if _, includeData, err := self.SplitTemplateHeaderContent(file); err == nil {
					if stat, err := file.Stat(); err == nil {
						log.Debugf("Injecting included template %q from file %s", name, stat.Name())

						define := "{{/* BEGIN INCLUDE '" + includePath + "' */}}\n"
						define += "{{ define \"" + name + "\" }}\n"
						end := "\n{{ end }}\n"
						end += "{{/* END INCLUDE '" + includePath + "' */}}\n"

						w.Write([]byte(define))
						w.Write(includeData)
						w.Write([]byte(end))
					} else {
						return err
					}
				} else {
					return err
				}
			} else {
				log.Debugf("Failed to open %q: %v", includePath, err)
			}
		}

		return nil
	}

	return nil
}

func (self *Server) setupServer() error {
	self.server = negroni.New()

	// setup panic recovery handler
	self.server.Use(negroni.NewRecovery())

	// setup internal/metadata routes
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("%s/_diecast", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc(fmt.Sprintf("%s/_bindings", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self.Bindings); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// all other routes proxy to this http.Handler
	mux.HandleFunc(fmt.Sprintf("%s/", self.RoutePrefix), self.handleFileRequest)

	self.server.UseHandler(mux)

	return nil
}

func requestToEvalData(req *http.Request, header *TemplateHeader) map[string]interface{} {
	rv := make(map[string]interface{})
	request := make(map[string]interface{})
	qs := make(map[string]interface{})
	hdr := make(map[string]interface{})

	// query strings
	// ------------------------------------------------------------------------
	if header != nil {
		for dK, dV := range header.Defaults {
			qs[dK] = stringutil.Autotype(dV)
		}
	}

	for k, v := range req.URL.Query() {
		if vv := strings.Join(v, `, `); !typeutil.IsZero(vv) {
			qs[k] = stringutil.Autotype(vv)
		}
	}

	// response headers
	// ------------------------------------------------------------------------
	if header != nil {
		for dK, dV := range header.DefaultHeaders {
			hdr[dK] = stringutil.Autotype(dV)
		}
	}

	for k, v := range req.Header {
		if vv := strings.Join(v, `, `); !typeutil.IsZero(vv) {
			hdr[k] = stringutil.Autotype(vv)
		}
	}

	request[`method`] = req.Method
	request[`protocol`] = req.Proto
	request[`headers`] = hdr
	request[`length`] = req.ContentLength
	request[`encoding`] = req.TransferEncoding
	request[`remote_address`] = req.RemoteAddr
	request[`host`] = req.Host

	url := map[string]interface{}{
		`unmodified`: req.RequestURI,
		`string`:     req.URL.String(),
		`scheme`:     req.URL.Scheme,
		`host`:       req.URL.Host,
		`path`:       req.URL.Path,
		`fragment`:   req.URL.Fragment,
		`query`:      qs,
	}

	if header != nil {
		url[`params`] = header.UrlParams
	}

	request[`url`] = url

	rv[`request`] = request

	return rv
}
