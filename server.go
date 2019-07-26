package diecast

//go:generate esc -o static.go -pkg diecast -modtime 1500000000 -prefix ui ui

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
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
	"github.com/ghetzel/go-stockutil/executil"
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
	"github.com/husobee/vestigo"
	"github.com/jbenet/go-base58"
	"github.com/mattn/go-shellwords"
	"github.com/urfave/negroni"
	"golang.org/x/text/language"
)

var ITotallyUnderstandRunningArbitraryCommandsAsRootIsRealRealBad = false
var DirectoryErr = errors.New(`is a directory`)
var DefaultLocale = language.AmericanEnglish

func IsDirectoryErr(err error) bool {
	return (err == DirectoryErr)
}

const DefaultAddress = `127.0.0.1:28419`
const DefaultRoutePrefix = `/`
const DefaultConfigFile = `diecast.yml`
const DefaultLayoutsPath = `/_layouts`
const DefaultErrorsPath = `/_errors`
const DebuggingQuerystringParam = `__viewsource`
const LayoutTemplateName = `layout`
const ContentTemplateName = `content`
const ContextRequestKey = `diecast-request-id`

var HeaderSeparator = []byte{'-', '-', '-'}
var DefaultIndexFile = `index.html`
var DefaultVerifyFile = `/` + DefaultIndexFile
var DefaultTemplatePatterns = []string{`*.html`, `*.md`, `*.scss`}
var DefaultTryExtensions = []string{`html`, `md`}
var DefaultAutoindexFilename = `/autoindex.html`

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
	// Exposes the location of the diecast binary
	BinPath string `json:"-"`

	// The host:port address the server is listening on
	Address string `json:"address"`

	// Top-level bindings that apply to every rendered template
	Bindings []Binding `json:"bindings"`

	// Specify a string to prefix all binding resource values that start with "/"
	BindingPrefix string `json:"bindingPrefix"`

	// The filesystem location where templates and files are served from
	RootPath string `json:"root"`

	// The path to the layouts template directory
	LayoutPath string `json:"layouts"`

	// The path to the errors template directory
	ErrorsPath string `json:"errors"`

	// Enables additional options for debugging applications. Caution: can expose secrets and other sensitive data.
	EnableDebugging bool `json:"debug"`

	// Disable emitting per-request Server-Timing headers to aid in tracing bottlenecks and performance issues.
	DisableTimings bool `json:"disableTimings"`

	// Specifies whether layouts are enabled
	EnableLayouts bool `json:"enableLayouts"`

	// If specified, all requests must be prefixed with this string.
	RoutePrefix string `json:"routePrefix"`

	// A set of glob patterns specifying which files will be rendered as templates.
	TemplatePatterns []string `json:"patterns"`

	// Allow for the programmatic addition of extra functions for use in templates.
	AdditionalFunctions template.FuncMap `json:"-"`

	// Whether to attempt to locate a local file matching the requested path before attempting to find a template.
	TryLocalFirst bool `json:"localFirst"`

	// The name of the template file to use when a directory is requested.
	IndexFile string `json:"indexFile"`

	// A file that must exist and be readable before starting the server.
	VerifyFile string `json:"verifyFile"`

	// The set of all registered mounts.
	Mounts []Mount `json:"-"`

	// A list of mount configurations read from the diecast.yml config file.
	MountConfigs []MountConfig `json:"mounts"`

	// A default header that all templates will inherit from.
	BaseHeader         *TemplateHeader        `json:"header"`
	DefaultPageObject  map[string]interface{} `json:"-"`
	OverridePageObject map[string]interface{} `json:"-"`

	// A command that will be executed before the server is started.
	PrestartCommands []*StartCommand `json:"prestart"`

	// A command that will be executed after the server is confirmed running.
	StartCommands []*StartCommand `json:"start"`

	// Disable the execution of PrestartCommands and StartCommand .
	DisableCommands bool `json:"disable_commands"`

	// A set of authenticator configurations used to protect some or all routes.
	Authenticators AuthenticatorConfigs `json:"authenticators"`

	// Try these file extensions when looking for default (i.e.: "index") files.  If IndexFile has an extension, it will be stripped first.
	TryExtensions []string `json:"tryExtensions"`

	// Map file extensions to preferred renderers for a given file type.
	RendererMappings map[string]string `json:"rendererMapping"`

	// Which types of files will automatically have layouts applied.
	AutolayoutPatterns []string `json:"autolayoutPatterns"`

	// List of filenames containing PEM-encoded X.509 TLS certificates that represent trusted authorities.
	// Use to validate certificates signed by an internal, non-public authority.
	TrustedRootPEMs []string `json:"trustedRootPEMs"`

	// Configure routes and actions to execute when those routes are requested.
	Actions []*Action `json:"actions"`

	// Specify that requests that terminate at a filesystem directory should automatically generate an index
	// listing of that directory.
	Autoindex bool `json:"autoindex"`

	// If Autoindex is enabled, this allows the template used to generate the index page to be customized.
	AutoindexTemplate string `json:"autoindex_template"`

	// Setup global configuration details for Binding Protocols
	Protocols map[string]ProtocolConfig `json:"protocols"`

	// A function that can be used to intercept handlers being added to the server.
	OnAddHandler AddHandlerFunc `json:"-"`

	// Stores translations for use with the i18n and l10n functions.  Keys values represent the
	Translations map[string]interface{} `json:"translations,omitempty"`

	// Specify the default locale for pages being served.
	Locale string `json:"locale"`

	router        *http.ServeMux
	userRouter    *vestigo.Router
	server        *negroni.Negroni
	fs            http.FileSystem
	precmd        *exec.Cmd
	altRootCaPool *x509.CertPool
	initialized   bool
	hasUserRoutes bool
}

func NewServer(root interface{}, patterns ...string) *Server {
	if len(patterns) == 0 {
		patterns = DefaultTemplatePatterns
	}

	describeTimer(`tpl`, `Diecast Template Rendering`)

	server := &Server{
		Address:            DefaultAddress,
		Authenticators:     make([]AuthenticatorConfig, 0),
		AutolayoutPatterns: DefaultAutolayoutPatterns,
		Bindings:           make([]Binding, 0),
		DefaultPageObject:  make(map[string]interface{}),
		EnableLayouts:      true,
		ErrorsPath:         DefaultErrorsPath,
		IndexFile:          DefaultIndexFile,
		LayoutPath:         DefaultLayoutsPath,
		Mounts:             make([]Mount, 0),
		OverridePageObject: make(map[string]interface{}),
		RendererMappings:   DefaultRendererMappings,
		RootPath:           `.`,
		RoutePrefix:        DefaultRoutePrefix,
		TemplatePatterns:   patterns,
		TryExtensions:      DefaultTryExtensions,
		VerifyFile:         DefaultVerifyFile,
		AutoindexTemplate:  DefaultAutoindexFilename,
		router:             http.NewServeMux(),
		userRouter:         vestigo.NewRouter(),
	}

	if str, ok := root.(string); ok {
		server.RootPath = str
	} else if fs, ok := root.(http.FileSystem); ok {
		server.SetFileSystem(fs)
	}

	server.router.HandleFunc(server.rp()+`/`, server.handleRequest)

	return server
}

func (self *Server) ShouldReturnSource(req *http.Request) bool {
	if self.EnableDebugging {
		if httputil.QBool(req, DebuggingQuerystringParam) {
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

// Append the specified mounts to the current server.
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

func (self *Server) IsInRootPath(path string) bool {
	if absR, err := filepath.Abs(self.RootPath); err == nil {
		if absP, err := filepath.Abs(path); err == nil {
			absR, _ := filepath.EvalSymlinks(absR)
			absP, _ := filepath.EvalSymlinks(absP)

			if absP == absR || strings.HasPrefix(absP, absR) {
				return true
			}
		}
	}

	return false
}

func (self *Server) Initialize() error {
	if v, err := fileutil.ExpandUser(self.RootPath); err == nil {
		self.RootPath = v
	}

	if v, err := filepath.Abs(self.RootPath); err == nil {
		self.RootPath = v
	} else {
		return fmt.Errorf("root path: %v", err)
	}

	// if we haven't explicitly set a filesystem, create it
	if self.fs == nil {
		self.SetFileSystem(http.Dir(self.RootPath))
	}

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

	// if configured, this path must exist (relative to RootPath or the root filesystem) or Diecast will refuse to start
	if self.VerifyFile != `` {
		if verify, err := self.fs.Open(self.VerifyFile); err == nil {
			verify.Close()
		} else {
			return fmt.Errorf("Failed to open verification file %q: %v.", self.VerifyFile, err)
		}
	}

	if err := self.setupServer(); err != nil {
		return err
	}

	self.initialized = true

	if self.DisableCommands {
		log.Noticef("Not executing PrestartCommand because DisableCommands is set")
		return nil
	} else if _, err := self.RunStartCommand(self.PrestartCommands, false); err != nil {
		return err
	} else {
		return nil
	}
}

func (self *Server) Serve() error {
	if self.server == nil {
		if err := self.Initialize(); err != nil {
			return err
		}
	}

	go func() {
		if self.DisableCommands {
			log.Noticef("Not executing StartCommand because DisableCommands is set")
			return
		}

		eoc, err := self.RunStartCommand(self.StartCommands, true)

		if eoc {
			defer func() {
				self.cleanupCommands()
				os.Exit(0)
			}()
		}

		if err != nil {
			log.Errorf("start command failed: %v", err)
		}
	}()

	return http.ListenAndServe(self.Address, self.server)
}

func (self *Server) ListenAndServe(address string) error {
	self.Address = address
	return self.Serve()
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if self.server == nil {
		if err := self.Initialize(); err != nil {
			w.Write([]byte(fmt.Sprintf("Failed to setup Diecast server: %v", err)))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	self.server.ServeHTTP(w, req)
}

// return whether the request path matches any of the configured TemplatePatterns.
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

// return whether the request path should automatically have layouts applied
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

// render a template, write the output to the given ResponseWriter
func (self *Server) applyTemplate(
	w http.ResponseWriter,
	req *http.Request,
	requestPath string,
	data []byte,
	header *TemplateHeader,
	urlParams map[string]interface{},
	mimeType string,
) error {
	fragments := make(FragmentSet, 0)
	forceSkipLayout := false
	layouts := make([]string, 0)

	// start building headers stack and calculate line offsets (for error reporting)
	if header != nil {
		if header.Layout != `` {
			if header.Layout == `false` || header.Layout == `none` {
				forceSkipLayout = true
			} else {
				layouts = []string{header.Layout}
			}
		}

		// add all includes from the current item
		if err := self.appendIncludes(&fragments, header); err != nil {
			return err
		}
	}

	earlyData := requestToEvalData(req, header)

	// get a reference to a set of standard functions that won't have a scope yet
	earlyFuncs := self.GetTemplateFunctions(earlyData, header)

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
						if err := fragments.Parse(LayoutTemplateName, layoutFile); err != nil {
							return err
						}

						break
					} else if layoutName != `default` {
						// we don't care if the default layout is missing
						return err
					}
				}
			}
		}
	}

	// get the content template in place
	// NOTE: make SURE this happens after the layout is loaded. this ensures that the layout data
	//       and bindings are evaluated first, then are overridden/appended by the content data/bindings
	if err := fragments.Set(ContentTemplateName, header, data); err != nil {
		return err
	}

	// get the merged header from all layouts, includes, and the template we're rendering
	finalHeader := fragments.Header(self)

	// add all includes
	if err := self.appendIncludes(&fragments, &finalHeader); err != nil {
		return err
	}

	// put any url route params in there too
	finalHeader.UrlParams = urlParams

	// render locale from template
	finalHeader.Locale = EvalInline(finalHeader.Locale, earlyData, earlyFuncs)

	if funcs, data, err := self.GetTemplateData(req, &finalHeader); err == nil {
		start := time.Now()

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
								log.Debugf("[%s] Switch case %d matched, switching to template %v", reqid(req), i, swcase.UsePath)
								// log.Dump(fh)

								return self.applyTemplate(
									w,
									req,
									requestPath,
									swData,
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

		var postTemplateRenderer Renderer
		var renderOpts = RenderOptions{
			FunctionSet:   funcs,
			Header:        &finalHeader,
			Fragments:     fragments,
			Data:          data,
			MimeType:      mimeType,
			RequestedPath: requestPath,
		}

		// if specified, get the FINAL renderer that the template output will be passed to
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
					log.Debugf("[%s] Rendering using %T", reqid(req), postTemplateRenderer)

					postTemplateRenderer.SetPrewriteFunc(func(r *http.Request) {
						reqtime(r, `tpl`, time.Since(start))
						writeRequestTimerHeaders(self, w, r)
					})

					return postTemplateRenderer.Render(w, req, renderOpts)
				} else {
					return err
				}
			} else {
				// just render the base template directly to the response and return

				baseRenderer.SetPrewriteFunc(func(r *http.Request) {
					reqtime(r, `tpl`, time.Since(start))
					writeRequestTimerHeaders(self, w, r)
				})

				return baseRenderer.Render(w, req, renderOpts)
			}
		} else {
			return err
		}
	} else if redir, ok := err.(RedirectTo); ok {
		log.Infof("[%s] Performing 307 Temporary Redirect to %v due to binding response handler.", reqid(req), redir)
		writeRequestTimerHeaders(self, w, req)
		http.Redirect(w, req, redir.Error(), http.StatusTemporaryRedirect)
		return nil
	} else {
		return err
	}
}

// Retrieves the set of standard template functions, as well as functions for working
// with data in the current request.
func (self *Server) GetTemplateFunctions(data interface{}, header *TemplateHeader) FuncMap {
	funcs := make(FuncMap)

	for k, v := range GetStandardFunctions(self) {
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

	// fn i18n: Return the translated text corresponding to the given key.
	//	Order of Preference:
	//	- Explicitly requested locale via the second argument to this function
	//  - Locale specified in the template header or parent headers
	//	- specified via the Accept-Language HTTP request header
	//	- Global server config (via diecast.yml "locale" setting)
	//	- Values of the LC_ALL, LANG, and LANGUAGE environment variables
	//	- compile-time default locale
	funcs[`i18n`] = func(key string, locales ...string) (string, error) {
		key = strings.Join(strings.Split(key, `.`), `.`)
		kparts := strings.Split(key, `.`)

		if header != nil && header.Locale != `` {
			if tag, err := language.Parse(header.Locale); err == nil {
				// header locale and country
				locales = append(locales, tag.String())
				locales = append(locales, i18nTagBase(tag))
			} else {
				log.Warningf("i18n: invalid header locale %q", header.Locale)
			}
		}

		// add server global preferred locale and country
		if self.Locale != `` {
			if tag, err := language.Parse(self.Locale); err == nil {
				locales = append(locales, tag.String())
				locales = append(locales, i18nTagBase(tag))
			} else {
				log.Warningf("i18n: invalid global locale %q", self.Locale)
			}
		}

		// add user-preferred languages via Accept-Language header
		if al := typeutil.String(maputil.DeepGet(data, []string{
			`request`,
			`headers`,
			`accept_language`,
		}, ``)); al != `` {
			if tags, _, err := language.ParseAcceptLanguage(al); err == nil {
				for _, tag := range tags {
					locales = append(locales, tag.String())
					locales = append(locales, i18nTagBase(tag))
				}
			} else {
				log.Warningf("i18n: invalid Accept-Language value %q", al)
			}
		}

		// add default locale and country
		locales = append(locales, DefaultLocale.String())
		locales = append(locales, i18nTagBase(DefaultLocale))

		// add values from environment variables
		for _, ev := range []string{
			`LC_ALL`,
			`LANG`,
			`LANGUAGE`,
		} {
			if v := os.Getenv(ev); v != `` {
				for _, localeEncodingPair := range strings.Split(v, `:`) {
					locale, _ := stringutil.SplitPair(localeEncodingPair, `.`)

					if tag, err := language.Parse(locale); err == nil {
						locales = append(locales, tag.String())
						locales = append(locales, i18nTagBase(tag))
					} else {
						log.Warningf("i18n: invalid locale in envvar %s", ev)
					}
				}
			}
		}

		locales = sliceutil.CompactString(locales)
		locales = sliceutil.UniqueStrings(locales)

		for _, translations := range []map[string]interface{}{
			header.Translations,
			self.Translations,
		} {
			for _, l := range locales {
				if t, ok := translations[string(l)]; ok {
					return typeutil.String(maputil.DeepGet(t, kparts, ``)), nil
				}
			}
		}

		return ``, fmt.Errorf("no translations available")
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
	return requestPath
}

func (self *Server) GetTemplateData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]interface{}, error) {
	data := requestToEvalData(req, header)

	data[`vars`] = make(map[string]interface{})

	data[`diecast`] = map[string]interface{}{
		`binding_prefix`:    self.BindingPrefix,
		`route_prefix`:      self.rp(),
		`template_patterns`: self.TemplatePatterns,
		`try_local_first`:   self.TryLocalFirst,
		`index_file`:        self.IndexFile,
		`verify_file`:       self.VerifyFile,
	}

	// these are the functions that will be available to every part of the rendering process
	funcs := self.GetTemplateFunctions(data, header)

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

	for i, binding := range bindingsToEval {
		if strings.TrimSpace(binding.Name) == `` {
			binding.Name = fmt.Sprintf("binding%d", i)
		}

		binding.server = self

		start := time.Now()
		describeTimer(fmt.Sprintf("binding-%s", binding.Name), fmt.Sprintf("Diecast Bindings: %s", binding.Name))

		if binding.Repeat == `` {
			bindings[binding.Name] = binding.Fallback
			data[`bindings`] = bindings

			v, err := binding.Evaluate(req, header, data, funcs)

			if err == nil && v != nil {
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

				v, err := binding.Evaluate(req, header, data, funcs)

				if err == nil {
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

		reqtime(req, fmt.Sprintf("binding-%s", binding.Name), time.Since(start))
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

func (self *Server) handleRequest(w http.ResponseWriter, req *http.Request) {
	id := reqid(req)
	prefix := fmt.Sprintf("%s/", self.rp())

	if strings.HasPrefix(req.URL.Path, prefix) {
		defer req.Body.Close()

		log.Infof("[%s] %v %v", id, req.Method, req.URL)
		requestPaths := []string{req.URL.Path}

		// if we're looking at a directory, throw in the index file if the path as given doesn't respond
		if strings.HasSuffix(req.URL.Path, `/`) {
			requestPaths = append(requestPaths, path.Join(req.URL.Path, self.IndexFile))

			for _, ext := range self.TryExtensions {
				base := filepath.Base(self.IndexFile)
				base = strings.TrimSuffix(base, filepath.Ext(self.IndexFile))

				requestPaths = append(requestPaths, path.Join(req.URL.Path, fmt.Sprintf("%s.%s", base, ext)))
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

		var triedLocal bool

	PathLoop:
		// search for the file in all of the generated request paths
		for _, rPath := range sliceutil.UniqueStrings(requestPaths) {
			// remove the Route Prefix, as that's a structural part of the path but does not
			// represent where the files are (used for embedding diecast in other services
			// to avoid name collisions)
			//
			rPath = strings.TrimPrefix(rPath, self.rp())

			var file http.File
			var statusCode int
			var mimeType string
			var redirectTo string
			var redirectCode int
			var headers = make(map[string]interface{})
			var urlParams = make(map[string]interface{})
			var forceTemplate bool

			if self.TryLocalFirst && !triedLocal {
				triedLocal = true

				// attempt loading the file from the local filesystem before searching the mounts
				if f, m, err := self.tryLocalFile(rPath, req); err == nil {
					file = f
					mimeType = m

				} else if IsDirectoryErr(err) && self.Autoindex {
					if f, m, ok := self.tryAutoindex(); ok {
						file = f
						mimeType = m
						forceTemplate = true
					} else {
						log.Warningf("[%s] failed to load autoindex template", id)
						continue
					}
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
				} else if IsDirectoryErr(err) && self.Autoindex {
					if f, m, ok := self.tryAutoindex(); ok {
						file = f
						mimeType = m
						forceTemplate = true
					} else {
						log.Warningf("[%s] failed to load autoindex template", id)
						continue
					}
				}
			}

			if redirectCode > 0 {
				if redirectTo == `` {
					redirectTo = fmt.Sprintf("%s/", req.URL.Path)
				}

				http.Redirect(w, req, redirectTo, redirectCode)
				log.Debugf("[%s]  path %v redirecting to %v (HTTP %d)", id, rPath, redirectTo, redirectCode)
				return
			}

			if file != nil {
				defer file.Close()

				if strings.Contains(rPath, `__id.`) {
					urlParams[`1`] = strings.Trim(path.Base(req.URL.Path), `/`)
					urlParams[`id`] = strings.Trim(path.Base(req.URL.Path), `/`)
				}

				if handled := self.tryToHandleFoundFile(rPath, mimeType, file, statusCode, headers, urlParams, w, req, forceTemplate); handled {
					return
				}
			}
		}
	}

	if self.hasUserRoutes {
		self.userRouter.ServeHTTP(w, req)
	} else {
		// if we got *here*, then File Not Found
		self.respondError(w, fmt.Errorf("[%s] File %q was not found.", id, req.URL.Path), http.StatusNotFound)
	}
}

func (self *Server) tryAutoindex() (http.File, string, bool) {
	if autoindex, err := self.fs.Open(self.AutoindexTemplate); err == nil {
		return autoindex, `text/html`, true
	} else if autoindex, err := FS(false).Open(self.AutoindexTemplate); err == nil {
		return autoindex, `text/html`, true
	} else {
		return nil, ``, false
	}
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
				return nil, ``, DirectoryErr
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
			}
		}
	}

	return nil, nil, fmt.Errorf("%q not found", requestPath)
}

func (self *Server) tryToHandleFoundFile(
	requestPath string,
	mimeType string,
	file http.File,
	statusCode int,
	headers map[string]interface{},
	urlParams map[string]interface{},
	w http.ResponseWriter,
	req *http.Request,
	forceTemplate bool,
) bool {
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
	if self.shouldApplyTemplate(requestPath) || forceTemplate {
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
			if err := self.applyTemplate(w, req, requestPath, templateData, header, urlParams, mimeType); err != nil {
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
		log.Warningf("%v (HTTP %d)", resErr, code)
	} else {
		log.Errorf("%v (HTTP %d)", resErr, code)
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

func (self *Server) appendIncludes(fragments *FragmentSet, header *TemplateHeader) error {
	if header != nil {
		for name, includePath := range header.Includes {
			if includeFile, err := self.fs.Open(includePath); err == nil {
				defer includeFile.Close()

				log.Debugf("Include template %q from file %s", name, includePath)
				fragments.Parse(name, includeFile)
			} else {
				return err
			}
		}
	}

	return nil
}

func reqid(req *http.Request) string {
	if id := req.Context().Value(ContextRequestKey); id != nil {
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
		log.Debugf("[%s] %s %s", requestId, req.Method, req.URL.Path)

		parent := req.Context()
		identified := context.WithValue(parent, ContextRequestKey, requestId)
		*req = *req.WithContext(identified)

		// setup request tracing info
		startRequestTimer(req)
	})

	// process authenticators
	self.server.UseFunc(func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
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

		// fallback to proceeding down the middleware chain
		next(w, req)
	})

	self.router.HandleFunc(fmt.Sprintf("%s/_diecast", self.rp()), func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
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
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	self.router.HandleFunc(fmt.Sprintf("%s/_bindings", self.rp()), func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
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
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	// add action handlers
	for i, action := range self.Actions {
		hndPath := filepath.Join(self.rp(), action.Path)

		if executil.IsRoot() && !executil.EnvBool(`DIECAST_ALLOW_ROOT_ACTIONS`) {
			return fmt.Errorf("Refusing to start as root with actions specified.  Override with the environment variable DIECAST_ALLOW_ROOT_ACTIONS=true")
		}

		if action.Path == `` {
			return fmt.Errorf("Action %d: Must specify a 'path'", i)
		}

		self.router.HandleFunc(hndPath, func(w http.ResponseWriter, req *http.Request) {
			if handler := self.actionForRequest(req); handler != nil {
				handler(w, req)
			} else {
				http.Error(w, fmt.Sprintf("cannot find handler for action"), http.StatusInternalServerError)
			}
		})

		log.Debugf("[actions] Registered %s", hndPath)
	}

	self.server.UseHandler(self.router)

	// cleanup request tracing info
	self.server.UseHandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		removeRequestTimer(req)
	})

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

	return nil
}

func (self *Server) actionForRequest(req *http.Request) http.HandlerFunc {
	route := req.URL.Path

	for _, action := range self.Actions {
		actionPath := filepath.Join(self.rp(), action.Path)

		if actionPath == route {
			methods := sliceutil.Stringify(action.Method)

			if len(methods) == 0 && req.Method == http.MethodGet {
				log.Debugf("Action handler: %s %s", http.MethodGet, action.Path)
				return action.ServeHTTP
			} else {
				for _, method := range methods {
					if req.Method == strings.ToUpper(method) {
						log.Debugf("Action handler: %s %s", req.Method, action.Path)
						return action.ServeHTTP
					}
				}
			}
		}
	}

	return nil
}

func (self *Server) rp() string {
	return strings.TrimSuffix(self.RoutePrefix, `/`)
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
			dK = stringutil.Underscore(strings.ToLower(dK))
			hdr[dK] = stringutil.Autotype(dV)
		}
	}

	for k, v := range req.Header {
		if vv := strings.Join(v, `, `); !typeutil.IsZero(vv) {
			k = stringutil.Underscore(strings.ToLower(k))
			hdr[k] = stringutil.Autotype(vv)
		}
	}

	request[`id`] = reqid(req)
	request[`timestamp`] = time.Now().UnixNano()
	request[`method`] = req.Method
	request[`protocol`] = req.Proto
	request[`headers`] = hdr
	request[`length`] = req.ContentLength

	if te := req.TransferEncoding; te == nil {
		request[`encoding`] = []string{`identity`}
	} else {
		request[`encoding`] = te
	}

	addr, port := stringutil.SplitPairRight(req.RemoteAddr, `:`)

	request[`remote_ip`] = addr
	request[`remote_port`] = int(typeutil.Int(port))
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
	} else {
		url[`params`] = make(map[string]interface{})
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

func (self *Server) RunStartCommand(scmds []*StartCommand, waitForCommand bool) (bool, error) {
	for _, scmd := range scmds {
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
				env[`DIECAST_ROUTE_PREFIX`] = self.rp()

				for key, value := range env {
					scmd.cmd.Env = append(scmd.cmd.Env, fmt.Sprintf("%v=%v", key, value))
				}

				if dir := scmd.Directory; dir != `` {
					if xdir, err := pathutil.ExpandUser(dir); err == nil {
						if absdir, err := filepath.Abs(xdir); err == nil {
							scmd.cmd.Dir = absdir
						} else {
							return false, err
						}
					} else {
						return false, err
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

					var xerr error

					if waitForCommand {
						xerr = <-waitchan
					}

					if xerr != nil || scmd.ExitOnCompletion {
						return scmd.ExitOnCompletion, xerr
					}
				} else {
					return false, err
				}
			} else {
				return false, fmt.Errorf("invalid command: %v", err)
			}
		}
	}

	return false, nil
}

func (self *Server) cleanupCommands() {
	for _, psc := range self.PrestartCommands {
		if psc.cmd != nil {
			if proc := psc.cmd.Process; proc != nil {
				proc.Kill()
			}
		}
	}

	for _, sc := range self.StartCommands {
		if sc.cmd != nil {
			if proc := sc.cmd.Process; proc != nil {
				proc.Kill()
			}
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

func i18nTagBase(tag language.Tag) string {
	if base, c := tag.Base(); c > language.Low {
		return base.String()
	} else {
		return ``
	}
}
