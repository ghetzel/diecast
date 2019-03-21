package diecast

//go:generate esc -o static.go -pkg diecast -modtime 1500000000 -prefix ui ui

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/structs"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/netutil"
	"github.com/ghetzel/go-stockutil/pathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghodss/yaml"
	"github.com/jbenet/go-base58"
	"github.com/julienschmidt/httprouter"
	"github.com/mattn/go-shellwords"
	"github.com/urfave/negroni"
)

const DefaultAddress = `127.0.0.1:28419`
const DefaultRoutePrefix = `/`
const DefaultConfigFile = `diecast.yml`

var HeaderSeparator = []byte{'-', '-', '-'}
var DefaultIndexFile = `index.html`
var DefaultVerifyFile = `/` + DefaultIndexFile
var DefaultTemplatePatterns = []string{`*.html`, `*.md`, `*.scss`}
var DefaultTryExtensions = []string{`html`, `md`}

var DefaultAutolayoutPatterns = []string{
	`*.html`,
	`*.md`,
}

var DefaultRendererMappings = map[string]string{
	`md`:   `markdown`,
	`scss`: `sass`,
}

type RedirectTo string

func (self RedirectTo) Error() string {
	return string(self)
}

type StartCommand struct {
	Command          string                 `json:"command"`
	Directory        string                 `json:"directory"`
	Environment      map[string]interface{} `json:"env"`
	WaitBefore       string                 `json:"delay"`
	Wait             string                 `json:"timeout"`
	ExitOnCompletion bool                   `json:"exitOnCompletion"`
	cmd              *exec.Cmd
}

type Server struct {
	BinPath             string                 `json:"-"`
	Address             string                 `json:"address"`
	Bindings            []Binding              `json:"bindings"`
	BindingPrefix       string                 `json:"bindingPrefix"`
	RootPath            string                 `json:"root"`
	LayoutPath          string                 `json:"layouts"`
	ErrorsPath          string                 `json:"errors"`
	EnableDebugging     bool                   `json:"debug"`
	EnableLayouts       bool                   `json:"enableLayouts"`
	RoutePrefix         string                 `json:"routePrefix"`
	TemplatePatterns    []string               `json:"patterns"`
	AdditionalFunctions template.FuncMap       `json:"-"`
	TryLocalFirst       bool                   `json:"localFirst"`
	IndexFile           string                 `json:"indexFile"`
	VerifyFile          string                 `json:"verifyFile"`
	Mounts              []Mount                `json:"-"`
	MountConfigs        []MountConfig          `json:"mounts"`
	BaseHeader          *TemplateHeader        `json:"header"`
	DefaultPageObject   map[string]interface{} `json:"-"`
	OverridePageObject  map[string]interface{} `json:"-"`
	PrestartCommand     StartCommand           `json:"prestart"`
	StartCommand        StartCommand           `json:"start"`
	Authenticators      AuthenticatorConfigs   `json:"authenticators"`
	TryExtensions       []string               `json:"tryExtensions"`   // try these file extensions when looking for default (i.e.: "index") files
	RendererMappings    map[string]string      `json:"rendererMapping"` // map file extensions to preferred renderers
	AutolayoutPatterns  []string               `json:"autolayoutPatterns"`
	TrustedRootPEMs     []string               `json:"trustedRootPEMs"`
	router              *httprouter.Router
	server              *negroni.Negroni
	fs                  http.FileSystem
	fsIsSet             bool
	fileServer          http.Handler
	precmd              *exec.Cmd
	altRootCaPool       *x509.CertPool
}

func NewServer(root string, patterns ...string) *Server {
	if len(patterns) == 0 {
		patterns = DefaultTemplatePatterns
	}

	return &Server{
		Address:            DefaultAddress,
		RoutePrefix:        DefaultRoutePrefix,
		DefaultPageObject:  make(map[string]interface{}),
		OverridePageObject: make(map[string]interface{}),
		Authenticators:     make([]AuthenticatorConfig, 0),
		RootPath:           root,
		EnableLayouts:      true,
		Bindings:           make([]Binding, 0),
		TemplatePatterns:   patterns,
		IndexFile:          DefaultIndexFile,
		VerifyFile:         DefaultVerifyFile,
		Mounts:             make([]Mount, 0),
		TryExtensions:      DefaultTryExtensions,
		RendererMappings:   DefaultRendererMappings,
		AutolayoutPatterns: DefaultAutolayoutPatterns,
	}
}

func (self *Server) ShouldReturnSource(req *http.Request) bool {
	if self.EnableDebugging {
		if httputil.QBool(req, `__viewsource`) {
			return true
		}
	}

	return false
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

	// allocate ephemeral address if we're supposed to
	if addr, port, err := net.SplitHostPort(self.Address); err == nil {
		if port == `0` {
			if allocated, err := netutil.EphemeralPort(); err == nil {
				self.Address = fmt.Sprintf("%v:%d", addr, allocated)
			} else {
				return err
			}
		}
	}

	if self.VerifyFile != `` {
		if verify, err := self.fs.Open(self.VerifyFile); err == nil {
			verify.Close()
		} else {
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

	// if we're appending additional trusted certs (for Bindings and other internal HTTP clients)
	if len(self.TrustedRootPEMs) > 0 {
		// get the existing system CA bundle
		if syspool, err := x509.SystemCertPool(); err == nil {
			// append each cert
			for _, pemfile := range self.TrustedRootPEMs {
				// must be a readable PEM file
				if pem, err := fileutil.ReadAll(pemfile); err == nil {
					if !syspool.AppendCertsFromPEM(pem) {
						return fmt.Errorf("Failed to append certificate %s", pemfile)
					}
				} else {
					return fmt.Errorf("Failed to read certificate %s: %v", pemfile, err)
				}
			}

			// this is what http.Client.Transport.TLSClientConfig.RootCAs will become
			self.altRootCaPool = syspool
		} else {
			return fmt.Errorf("Failed to retrieve system CA pool: %v", err)
		}
	}

	return self.RunStartCommand(&self.PrestartCommand, false)
}

func (self *Server) Serve() error {
	go func() {
		if err := self.RunStartCommand(&self.StartCommand, true); err != nil {
			log.Errorf("start command failed: %v", err)

			if self.StartCommand.ExitOnCompletion {
				self.cleanupCommands()
				os.Exit(1)
			}
		} else if self.StartCommand.ExitOnCompletion {
			self.cleanupCommands()
			os.Exit(0)
		}
	}()

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

func (self *Server) shouldApplyLayout(requestPath string) bool {
	baseName := filepath.Base(requestPath)

	for _, pattern := range self.AutolayoutPatterns {
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
			headerOffset = header.lines
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

	// get a reference to a set of standard functions that won't have a scope yet
	earlyFuncs := self.GetTemplateFunctions(requestToEvalData(req, header))

	// only process layouts if we're supposed to
	if self.EnableLayouts && !forceSkipLayout && self.shouldApplyLayout(requestPath) {
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
					layoutName = EvalInline(layoutName, nil, earlyFuncs)

					if layoutFile, err := self.LoadLayout(layoutName); err == nil {
						if layoutHeader, layoutData, err := SplitTemplateHeaderContent(layoutFile); err == nil {
							if layoutHeader != nil {
								headers = append([]*TemplateHeader{layoutHeader}, headers...)

								// add in layout includes
								if err := self.InjectIncludes(finalTemplate, layoutHeader); err != nil {
									return err
								}
							}

							finalTemplate.WriteString("{{/* BEGIN LAYOUT '" + layoutName + "' */}}\n")
							appendTemplate(finalTemplate, bytes.NewBuffer(layoutData), `layout`, true)
							finalTemplate.WriteString("{{/* END LAYOUT '" + layoutName + "' */}}\n\n")
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
		// switches allow the template processing to be hijacked/redirected mid-evaluation
		// based on data already evaluated
		if len(finalHeader.Switch) > 0 {
			for i, swcase := range finalHeader.Switch {
				if swcase == nil {
					continue
				}

				if swcase.UsePath != `` {
					// if a condition is specified, it must evalutate to a truthy value to proceed
					if swcase.Condition != `` {
						if !typeutil.V(EvalInline(swcase.Condition, data, funcs)).Bool() {
							continue
						}
					}

					if swTemplate, err := self.fs.Open(swcase.UsePath); err == nil {
						if swHeader, swData, err := SplitTemplateHeaderContent(swTemplate); err == nil {
							finalHeader.Switch[i] = nil

							if fh, err := finalHeader.Merge(swHeader); err == nil {
								log.Debugf("Switch case %d matched, switching to template %v", i, swcase.UsePath)
								// log.Dump(fh)

								return self.applyTemplate(
									w,
									req,
									requestPath,
									bytes.NewBuffer(swData),
									fh,
									urlParams,
									mimeType,
								)
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
			}
		}

		// append the template to the final output (which at this point may or may not
		// include the layout and explicitly-included subtemplates/snippets)
		if err := appendTemplate(finalTemplate, reader, `content`, hasLayout); err != nil {
			return err
		}

		var postTemplateRenderer Renderer
		var renderOpts = RenderOptions{
			FunctionSet:  funcs,
			HasLayout:    hasLayout,
			Header:       finalHeader,
			HeaderOffset: headerOffset,
			Input: ioutil.NopCloser(
				bytes.NewReader(finalTemplate.Bytes()),
			),
			Data:          data,
			MimeType:      mimeType,
			RequestedPath: requestPath,
		}

		// if specified, get the FINAL renderer that the template output will be passed to
		if finalHeader != nil {
			finalHeader.Renderer = EvalInline(finalHeader.Renderer, data, funcs)

			switch finalHeader.Renderer {
			case ``, `html`:
				if r, ok := GetRendererForFilename(requestPath, self); ok {
					postTemplateRenderer = r
				}
			default:
				if r, err := GetRenderer(finalHeader.Renderer, self); err == nil {
					postTemplateRenderer = r
				} else {
					return err
				}
			}
		}

		// evaluate and render the template first
		if baseRenderer, err := GetRenderer(``, self); err == nil {
			// if a user-specified renderer was provided, take the rendered output and
			// pass it into that renderer.  return the result
			if postTemplateRenderer != nil {
				var err error

				if postTemplateRenderer.ShouldPrerender() || httputil.QBool(req, `__subrender`) {
					// we use an httptest.ResponseRecorder to intercept the default template's output
					// and pass it as input to the final renderer.
					intercept := httptest.NewRecorder()

					err = baseRenderer.Render(intercept, req, renderOpts)
					res := intercept.Result()
					renderOpts.MimeType = res.Header.Get(`Content-Type`)
					renderOpts.Input = res.Body
				}

				if err == nil {
					// run the final template render and return
					log.Debugf("Rendering using %T", postTemplateRenderer)
					return postTemplateRenderer.Render(w, req, renderOpts)
				} else {
					return err
				}
			} else {
				// just render the base template directly to the response and return
				return baseRenderer.Render(w, req, renderOpts)
			}
		} else {
			return err
		}
	} else if redir, ok := err.(RedirectTo); ok {
		log.Infof("Performing 307 Temporary Redirect to %v due to binding response handler.", redir)
		http.Redirect(w, req, redir.Error(), http.StatusTemporaryRedirect)
		return nil
	} else {
		return err
	}
}

// Retrieves the set of standard template functions, as well as functions for working
// with data in the current request.
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
			value = nil
		case 1:
			value = vI[0]
		default:
			value = vI
		}

		maputil.DeepSet(data, makeVarKey(name), value)
		return ``
	}

	// fn varset: Treat the runtime variable *name* as a map, setting *key* to *value*.
	funcs[`varset`] = func(name string, key string, vI ...interface{}) interface{} {
		var value interface{}
		path := makeVarKey(name)

		switch len(vI) {
		case 0:
			value = make(map[string]interface{})
		case 1:
			value = vI[0]
		default:
			value = vI
		}

		maputil.DeepSet(data, append(path, strings.Split(key, `.`)...), value)
		return ``
	}

	// fn push: Append to variable *name* to *value*.
	funcs[`push`] = func(name string, vI ...interface{}) interface{} {
		var values []interface{}
		key := makeVarKey(name)

		if existing := maputil.DeepGet(data, key); existing != nil {
			values = append(values, sliceutil.Sliceify(existing)...)
		}

		values = append(values, vI...)
		maputil.DeepSet(data, key, values)

		return ``
	}

	// fn pop: Remove the last item from *name* and return it.
	funcs[`pop`] = func(name string) interface{} {
		var out interface{}
		key := makeVarKey(name)

		if existing := maputil.DeepGet(data, key); existing != nil {
			values := sliceutil.Sliceify(existing)

			switch len(values) {
			case 0:
				return nil
			case 1:
				out = values[0]
				maputil.DeepSet(data, key, nil)
			default:
				out = values[len(values)-1]
				values = values[0 : len(values)-1]
				maputil.DeepSet(data, key, values)
			}
		}

		return out
	}

	// fn increment: Increment a named variable by an amount.
	funcs[`increment`] = func(name string, incr ...int) interface{} {
		key := makeVarKey(name)
		count := 0

		if existing := maputil.DeepGet(data, key); existing != nil {
			count = int(typeutil.V(existing).Int())
		}

		if len(incr) > 0 {
			count += incr[0]
		} else {
			count += 1
		}

		maputil.DeepSet(data, key, count)

		return ``
	}

	// fn incrementByValue: Add a number to a counter tracking the number of occurrences of a specific value.
	funcs[`incrementByValue`] = func(name string, value interface{}, incr ...int) interface{} {
		key := makeVarKey(name, fmt.Sprintf("%v", value))
		count := 0

		if existing := maputil.DeepGet(data, key); existing != nil {
			count = int(typeutil.V(existing).Int())
		}

		if len(incr) > 0 {
			count += incr[0]
		} else {
			count += 1
		}

		maputil.DeepSet(data, key, count)

		return ``
	}

	// read a file from the serving path
	funcs[`read`] = func(filename string) (string, error) {
		if file, err := self.fs.Open(filename); err == nil {
			defer file.Close()

			if data, err := ioutil.ReadAll(file); err == nil {
				return string(data), nil
			} else {
				return ``, err
			}
		} else {
			return ``, err
		}
	}

	return funcs
}

func makeVarKey(key string, post ...string) []string {
	output := []string{`vars`}

	output = append(output, strings.Split(key, `.`)...)
	output = append(output, post...)

	return output
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

	data[`vars`] = make(map[string]interface{})

	data[`diecast`] = map[string]interface{}{
		`binding_prefix`:    self.BindingPrefix,
		`route_prefix`:      self.RoutePrefix,
		`template_patterns`: self.TemplatePatterns,
		`try_local_first`:   self.TryLocalFirst,
		`index_file`:        self.IndexFile,
		`verify_file`:       self.VerifyFile,
	}

	// these are the functions that will be available to every part of the rendering process
	funcs := self.GetTemplateFunctions(data)

	// Evaluate "page" data: this data is templatized, but does not have access
	//                       to the output of bindings
	// ---------------------------------------------------------------------------------------------
	if header != nil {
		pageData := make(map[string]interface{})

		applyPageFn := func(value interface{}, path []string, isLeaf bool) error {

			if isLeaf {
				switch value.(type) {
				case string:
					value = EvalInline(value.(string), data, funcs)
					value = stringutil.Autotype(value)
				}

				maputil.DeepSet(pageData, path, value)
			}

			return nil
		}

		// add default page object values
		maputil.Walk(self.DefaultPageObject, applyPageFn)

		// then pepper in whatever values came from the aggregated headers from
		// the layout, includes, and target template
		maputil.Walk(header.Page, applyPageFn)

		// if there were override items specified (e.g.: via the command line), add them now
		maputil.Walk(self.OverridePageObject, applyPageFn)

		data[`page`] = pageData
	} else {
		data[`page`] = make(map[string]interface{})
	}

	// Evaluate "bindings": Bindings have access to $.page, and each subsequent binding has access
	//                      to all binding output that preceded it.  This allows bindings to be
	//                      pipelined, using the output of one request as the input of the next.
	// ---------------------------------------------------------------------------------------------
	bindings := make(map[string]interface{})
	bindingsToEval := make([]Binding, 0)

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
			} else if redir, ok := err.(RedirectTo); ok {
				return funcs, nil, redir
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
				} else if redir, ok := err.(RedirectTo); ok {
					return funcs, nil, redir
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

	// Evaluate "flags" data: this data is templatized, and has access to $.page and $.bindings
	// ---------------------------------------------------------------------------------------------
	if header != nil {
		flags := make(map[string]bool)

		for name, def := range header.FlagDefs {
			switch def.(type) {
			case bool:
				flags[name] = def.(bool)
			default:
				flags[name] = typeutil.V(EvalInline(fmt.Sprintf("%v", def), data, funcs)).Bool()
			}
		}

		data[`flags`] = flags
	}

	return funcs, data, nil
}

func (self *Server) handleFileRequest(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	log.Infof("%v %v", req.Method, req.URL)

	if auth, err := self.Authenticators.Authenticator(req); err == nil {
		if auth != nil {
			if auth.IsCallback(req.URL) {
				auth.Callback(w, req)
				return
			} else if !auth.Authenticate(w, req) {
				return
			}
		}
	} else {
		self.respondError(w, err, http.StatusInternalServerError)
	}

	// normalize filename from request path
	requestPath := req.URL.Path

	requestPaths := []string{
		requestPath,
	}

	// if we're looking at a directory, throw in the index file if the path as given doesn't respond
	if strings.HasSuffix(requestPath, `/`) {
		requestPaths = append(requestPaths, path.Join(requestPath, self.IndexFile))

		for _, ext := range self.TryExtensions {
			base := filepath.Base(self.IndexFile)
			base = strings.TrimSuffix(base, filepath.Ext(self.IndexFile))

			requestPaths = append(requestPaths, path.Join(requestPath, fmt.Sprintf("%s.%s", base, ext)))
		}

	} else if path.Ext(requestPath) == `` {
		// if we're requesting a path without a file extension, try an index file in a directory with that name,
		// then try just <filename>.html
		requestPaths = append(requestPaths, fmt.Sprintf("%s/%s", requestPath, self.IndexFile))

		for _, ext := range self.TryExtensions {
			requestPaths = append(requestPaths, fmt.Sprintf("%s.%s", requestPath, ext))
		}
	}

	// finally, add handlers for implementing a junky form of url routing
	if parent := path.Dir(requestPath); parent != `.` {
		for _, ext := range self.TryExtensions {
			requestPaths = append(requestPaths, fmt.Sprintf("%s/index__id.%s", strings.TrimSuffix(parent, `/`), ext))

			if base := strings.TrimSuffix(parent, `/`); base != `` {
				requestPaths = append(requestPaths, fmt.Sprintf("%s__id.%s", base, ext))
			}
		}
	}

	var triedLocal bool

PathLoop:
	// search for the file in all of the generated request paths
	for _, rPath := range requestPaths {
		// remove the Route Prefix, as that's a structural part of the path but does not
		// represent where the files are (used for embedding diecast in other services
		// to avoid name collisions)
		//
		rPath = strings.TrimPrefix(rPath, self.RoutePrefix)
		var file http.File
		var statusCode int
		var mimeType string
		var redirectTo string
		var redirectCode int
		var headers = make(map[string]interface{})
		var urlParams = make(map[string]interface{})

		if self.TryLocalFirst && !triedLocal {
			triedLocal = true

			// attempt loading the file from the local filesystem before searching the mounts
			if f, m, err := self.tryLocalFile(rPath, req); err == nil {
				file = f
				mimeType = m

			} else if _, response, err := self.tryMounts(rPath, req); err == nil {
				file = response.GetFile()
				mimeType = response.ContentType
				statusCode = response.StatusCode
				headers = response.Metadata
				redirectTo = response.RedirectTo
				redirectCode = response.RedirectCode

			} else if IsHardStop(err) {
				break PathLoop
			}
		} else {
			// search the mounts before attempting to load the file from the local filesystem
			if _, response, err := self.tryMounts(rPath, req); err == nil && response != nil {
				file = response.GetFile()
				mimeType = response.ContentType
				statusCode = response.StatusCode
				headers = response.Metadata
				redirectTo = response.RedirectTo
				redirectCode = response.RedirectCode

			} else if IsHardStop(err) {
				break PathLoop

			} else if f, m, err := self.tryLocalFile(rPath, req); err == nil {
				file = f
				mimeType = m
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
			defer file.Close()

			if strings.Contains(rPath, `__id.`) {
				urlParams[`1`] = strings.Trim(path.Base(req.URL.Path), `/`)
				urlParams[`id`] = strings.Trim(path.Base(req.URL.Path), `/`)
			}

			if handled := self.tryToHandleFoundFile(rPath, mimeType, file, statusCode, headers, urlParams, w, req); handled {
				return
			}
		}
	}

	// if we got *here*, then File Not Found
	// log.Debugf("< not found")

	self.respondError(w, fmt.Errorf("File %q was not found.", requestPath), http.StatusNotFound)
}

// Attempt to resolve the given path into a real file and return that file and mime type.
// Non-existent files, unreadable files, and directories will return an error.
func (self *Server) tryLocalFile(requestPath string, req *http.Request) (http.File, string, error) {
	// if we got here, try to serve the file from the filesystem
	if file, err := self.fs.Open(requestPath); err == nil {
		if stat, err := file.Stat(); err == nil {
			if !stat.IsDir() {
				if mimetype := httputil.Q(req, `mimetype`); mimetype != `` {
					return file, mimetype, nil
				} else if mimetype, err := figureOutMimeType(stat.Name(), file); err == nil {
					return file, mimetype, nil
				} else {
					return file, ``, err
				}
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

// Try to load the given path from each of the mounts, and return the matching mount and its response
// if found.
func (self *Server) tryMounts(requestPath string, req *http.Request) (Mount, *MountResponse, error) {
	var body *bytes.Reader

	// buffer the request body because we need to repeatedly pass it to multiple mounts
	if data, err := ioutil.ReadAll(req.Body); err == nil {
		if len(data) > 0 {
			log.Debugf("  read %d bytes from request body\n", len(data))
		}

		body = bytes.NewReader(data)
	} else {
		return nil, nil, err
	}

	// find a mount that has this file
	for _, mount := range self.Mounts {
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
				log.Warningf("%v", err)
			}
		}
	}

	return nil, nil, fmt.Errorf("%q not found", requestPath)
}

func (self *Server) tryToHandleFoundFile(requestPath string, mimeType string, file http.File, statusCode int, headers map[string]interface{}, urlParams map[string]interface{}, w http.ResponseWriter, req *http.Request) bool {
	// add in any metadata as response headers
	for k, v := range headers {
		w.Header().Set(k, fmt.Sprintf("%v", v))
	}

	if mimeType == `` {
		mimeType = fileutil.GetMimeType(requestPath, `application/octet-stream`)
	}

	// write out the HTTP status if we were given one
	if statusCode > 0 {
		w.WriteHeader(statusCode)
	}

	// we got a real actual file here, figure out if we're templating it or not
	if self.shouldApplyTemplate(requestPath) {
		// tease the template header out of the file
		if header, templateData, err := SplitTemplateHeaderContent(file); err == nil {
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
		// if not templated, then the file is returned outright
		if rendererName := httputil.Q(req, `renderer`); rendererName == `` {
			w.Header().Set(`Content-Type`, mimeType)
			io.Copy(w, file)
		} else if renderer, err := GetRenderer(rendererName, self); err == nil {
			if err := renderer.Render(w, req, RenderOptions{
				Input: file,
			}); err != nil {
				self.respondError(w, err, http.StatusInternalServerError)
			}
		} else if renderer, ok := GetRendererForFilename(requestPath, self); ok {
			if err := renderer.Render(w, req, RenderOptions{
				Input: file,
			}); err != nil {
				self.respondError(w, err, http.StatusInternalServerError)
			}
		} else {
			self.respondError(w, fmt.Errorf("Unknown renderer %q", rendererName), http.StatusBadRequest)
		}
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

func SplitTemplateHeaderContent(reader io.Reader) (*TemplateHeader, []byte, error) {
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
			if includeFile, err := self.fs.Open(includePath); err == nil {
				defer includeFile.Close()

				if includeHeader, includeData, err := SplitTemplateHeaderContent(includeFile); err == nil {
					if stat, err := includeFile.Stat(); err == nil {
						log.Debugf("Injecting included template %q from file %s", name, stat.Name())

						// merge in included header
						if includeHeader != nil {
							if newHeader, err := header.Merge(includeHeader); err == nil {
								*header = *newHeader
							} else {
								return fmt.Errorf("include %v: %v", name, err)
							}
						}

						w.Write([]byte("{{/* BEGIN INCLUDE '" + includePath + "' */}}\n"))
						appendTemplate(w, bytes.NewBuffer(includeData), name, true)
						w.Write([]byte("{{/* END INCLUDE '" + includePath + "' */}}\n\n"))
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

func reqid(req *http.Request) string {
	if id := req.Context().Value(`diecast-request-id`); id != nil {
		return fmt.Sprintf("%v", id)
	} else {
		return ``
	}
}

func (self *Server) setupServer() error {
	self.server = negroni.New()

	// setup panic recovery handler
	self.server.Use(negroni.NewRecovery())

	// setup request ID generation
	self.server.UseHandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestId := base58.Encode(stringutil.UUID().Bytes())

		parent := req.Context()
		identified := context.WithValue(parent, `diecast-request-id`, requestId)
		*req = *req.WithContext(identified)
	})

	// setup internal/metadata routes
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("%s/_diecast", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		if req.Header.Get(`X-Diecast-Binding`) != `` {
			if data, err := json.Marshal(self); err == nil {
				w.Header().Set(`Content-Type`, `application/json`)

				if _, err := w.Write(data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, fmt.Sprintf("File %q was not found.", req.URL.Path), http.StatusNotFound)
		}
	})

	mux.HandleFunc(fmt.Sprintf("%s/_bindings", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()

		if req.Header.Get(`X-Diecast-Binding`) != `` {
			if data, err := json.Marshal(self.Bindings); err == nil {
				w.Header().Set(`Content-Type`, `application/json`)

				if _, err := w.Write(data); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, fmt.Sprintf("File %q was not found.", req.URL.Path), http.StatusNotFound)
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

	// environment variables
	env := make(map[string]interface{})

	for _, pair := range os.Environ() {
		key, value := stringutil.SplitPair(pair, `=`)
		env[key] = stringutil.Autotype(value)
	}

	rv[`env`] = env

	return rv
}

func (self *Server) RunStartCommand(scmd *StartCommand, waitForCommand bool) error {
	if cmdline := scmd.Command; cmdline != `` {
		if tokens, err := shellwords.Parse(cmdline); err == nil {
			scmd.cmd = exec.Command(tokens[0], tokens[1:]...)
			scmd.cmd.SysProcAttr = &syscall.SysProcAttr{
				Setpgid: true,
			}

			env := make(map[string]interface{})

			for _, pair := range os.Environ() {
				key, value := stringutil.SplitPair(pair, `=`)
				env[key] = value
			}

			for key, value := range scmd.Environment {
				env[key] = value
			}

			env[`DIECAST`] = true
			env[`DIECAST_BIN`] = self.BinPath
			env[`DIECAST_DEBUG`] = self.EnableDebugging
			env[`DIECAST_ADDRESS`] = self.Address
			env[`DIECAST_ROOT`] = self.RootPath
			env[`DIECAST_PATH_LAYOUTS`] = self.LayoutPath
			env[`DIECAST_PATH_ERRORS`] = self.ErrorsPath
			env[`DIECAST_BINDING_PREFIX`] = self.BindingPrefix
			env[`DIECAST_ROUTE_PREFIX`] = self.RoutePrefix

			for key, value := range env {
				scmd.cmd.Env = append(scmd.cmd.Env, fmt.Sprintf("%v=%v", key, value))
			}

			if dir := scmd.Directory; dir != `` {
				if xdir, err := pathutil.ExpandUser(dir); err == nil {
					if absdir, err := filepath.Abs(xdir); err == nil {
						scmd.cmd.Dir = absdir
					} else {
						return err
					}
				} else {
					return err
				}
			}

			if prewait, err := timeutil.ParseDuration(scmd.WaitBefore); err == nil && prewait > 0 {
				log.Infof("Waiting %v before running command", prewait)
				time.Sleep(prewait)
			}

			if wait, err := timeutil.ParseDuration(scmd.Wait); err == nil {
				waitchan := make(chan error)

				go func() {
					log.Infof("Executing command: %v", strings.Join(scmd.cmd.Args, ` `))
					waitchan <- scmd.cmd.Run()
				}()

				time.Sleep(wait)

				if waitForCommand {
					return <-waitchan
				} else {
					return nil
				}
			} else {
				return err
			}
		} else {
			return fmt.Errorf("invalid command: %v", err)
		}
	} else {
		return nil
	}
}

func (self *Server) cleanupCommands() {
	if self.PrestartCommand.cmd != nil {
		if proc := self.PrestartCommand.cmd.Process; proc != nil {
			proc.Kill()
		}
	}

	if self.StartCommand.cmd != nil {
		if proc := self.StartCommand.cmd.Process; proc != nil {
			proc.Kill()
		}
	}
}

func appendTemplate(dest io.Writer, src io.Reader, name string, hasLayout bool) error {
	if hasLayout {
		dest.Write([]byte("\n{{ define \"" + name + "\" }}\n"))
	}

	if _, err := io.Copy(dest, src); err != nil {
		return err
	}

	if hasLayout {
		dest.Write([]byte("\n{{ end }}\n"))
	}

	return nil
}
