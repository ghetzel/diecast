package diecast

//go:generate esc -o static.go -pkg diecast -modtime 1500000000 -prefix ui ui
//go:generate make favicon.go

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"html/template"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

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
	"github.com/gobwas/glob"
	"github.com/husobee/vestigo"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/mattn/go-shellwords"
	"github.com/signalsciences/tlstext"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

var ITotallyUnderstandRunningArbitraryCommandsAsRootIsRealRealBad = false
var DirectoryErr = errors.New(`is a directory`)
var DefaultLocale = language.AmericanEnglish
var DefaultLogFormat = `common`
var DefaultProtocol = `http`

var logFormats = map[string]string{
	`common`: "${remote_address} - - [${request_started_at:%02/Jan/2006:15:04:05 -0700}] \"${method} ${url} ${protocol}\" ${status_code} ${response_length}\n",
}

// Registers a new named format for log output
func RegisterLogFormat(name string, format string) {
	logFormats[name] = format
}

func IsDirectoryErr(err error) bool {
	return (err == DirectoryErr)
}

const DefaultQueryJoiner = `,`
const DefaultHeaderJoiner = `,`
const DefaultAddress = `127.0.0.1:28419`
const DefaultRoutePrefix = `/`
const DefaultConfigFile = `diecast.yml`
const DefaultLayoutsPath = `/_layouts`
const DefaultErrorsPath = `/_errors`
const DebuggingQuerystringParam = `__viewsource`
const LayoutTemplateName = `layout`
const ContentTemplateName = `content`
const ContextRequestKey = `diecast-request-id`
const ContextResponseKey = `diecast-response`

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

var DefaultFilterEnvVars = []string{
	`_*`,                    // treat vars starting with "_" as internal/hidden
	`*KEY*`,                 // omit api keys and whatnot
	`*PASSWORD*`,            // omit things that explicitly call themselves a "password"
	`*PID*`,                 // avoid PID leakage for weird paranoid reasons
	`*HOST*`,                // while we're at it, don't leak hostnames either
	`*URL*`,                 // ...or URLs
	`*SECRET*`,              // not a secret if you go blabbing about it...
	`*TOKEN*`,               // ditto.
	`AWS_ACCESS_KEY_ID`,     // very specifically omit AWS credentials
	`AWS_SECRET_ACCESS_KEY`, // very specifically omit AWS credentials
	`PROMPT_COMMAND`,        // people keep weird stuff in here sometimes
	`PWD`,                   // WHO WANTS TO KNOW?
}

type ServeFunc func(*Server) error

type Serveable interface {
	ListenAndServe() error
	ListenAndServeTLS(string, string) error
	Serve(net.Listener) error
	ServeTLS(net.Listener, string, string) error
}

type RedirectTo string

func (self RedirectTo) Error() string {
	return string(self)
}

type StartCommand struct {
	Command          string                 `yaml:"command"          json:"command"`          // The shell command line to execute on start
	Directory        string                 `yaml:"directory"        json:"directory"`        // The working directory the command should be run from
	Environment      map[string]interface{} `yaml:"env"              json:"env"`              // A map of environment variables to expose to the command
	WaitBefore       string                 `yaml:"delay"            json:"delay"`            // How long to delay before running the command
	Wait             string                 `yaml:"timeout"          json:"timeout"`          // How long to wait before killing the command
	ExitOnCompletion bool                   `yaml:"exitOnCompletion" json:"exitOnCompletion"` // Whether Diecast should exit upon command completion
	cmd              *exec.Cmd
}

type TlsConfig struct {
	Enable         bool   `yaml:"enable"   json:"enable"`   // Whether to enable SSL/TLS on the server.
	CertFile       string `yaml:"cert"     json:"cert"`     // path to a PEM-encoded (.crt) file containing the server's TLS public key.
	KeyFile        string `yaml:"key"      json:"key"`      // path to a PEM-encoded (.key) file containing the server's TLS private key.
	ClientCertMode string `yaml:"clients"  json:"clients"`  // If set, TLS Client certificates will be requested/accepted.  If set, may be one of: "request", "any", "verify", "require"
	ClientCAFile   string `yaml:"clientCA" json:"clientCA"` // Path to a PEM-encoded file containing the CA that client certificates are issued and verify against.
}

type LogConfig struct {
	Format      string `yaml:"format"               json:"format"`             // configure the output format for logging requests
	Destination string `yaml:"destination"                 json:"destination"` // specify where logs should be written to
	Truncate    bool   `yaml:"truncate"             json:"truncate"`           // if true, the output log file will be truncated on startup
	Colorize    bool   `yaml:"colorize"             json:"colorize"`           // if false, log output will not be colorized
}

type Server struct {
	Actions             []*Action                 `yaml:"actions"                 json:"actions"`                 // Configure routes and actions to execute when those routes are requested.
	AdditionalFunctions template.FuncMap          `yaml:"-"                       json:"-"`                       // Allow for the programmatic addition of extra functions for use in templates.
	Address             string                    `yaml:"address"                 json:"address"`                 // The host:port address the server is listening on
	Authenticators      AuthenticatorConfigs      `yaml:"authenticators"          json:"authenticators"`          // A set of authenticator configurations used to protect some or all routes.
	Autoindex           bool                      `yaml:"autoindex"               json:"autoindex"`               // Specify that requests that terminate at a filesystem directory should automatically generate an index listing of that directory.
	AutoindexTemplate   string                    `yaml:"autoindexTemplate"       json:"autoindexTemplate"`       // If Autoindex is enabled, this allows the template used to generate the index page to be customized.
	AutolayoutPatterns  []string                  `yaml:"autolayoutPatterns"      json:"autolayoutPatterns"`      // Which types of files will automatically have layouts applied.
	BaseHeader          *TemplateHeader           `yaml:"header"                  json:"header"`                  // A default header that all templates will inherit from.
	BinPath             string                    `yaml:"-"                       json:"-"`                       // Exposes the location of the diecast binary
	BindingPrefix       string                    `yaml:"bindingPrefix"           json:"bindingPrefix"`           // Specify a string to prefix all binding resource values that start with "/"
	Bindings            []Binding                 `yaml:"bindings"                json:"bindings"`                // Top-level bindings that apply to every rendered template
	DefaultPageObject   map[string]interface{}    `yaml:"-"                       json:"-"`                       //
	DisableCommands     bool                      `yaml:"disable_commands"        json:"disable_commands"`        // Disable the execution of PrestartCommands and StartCommand .
	DisableTimings      bool                      `yaml:"disableTimings"          json:"disableTimings"`          // Disable emitting per-request Server-Timing headers to aid in tracing bottlenecks and performance issues.
	EnableDebugging     bool                      `yaml:"debug"                   json:"debug"`                   // Enables additional options for debugging applications. Caution: can expose secrets and other sensitive data.
	DebugDumpRequests   map[string]string         `yaml:"debugDumpRequests"       json:"debugDumpRequests"`       // An object keyed on path globs whose values are a directory where matching requests are dumped in their entirety as text files.
	EnableLayouts       bool                      `yaml:"enableLayouts"           json:"enableLayouts"`           // Specifies whether layouts are enabled
	Environment         string                    `yaml:"environment"             json:"environment"`             // Specify the environment for loading environment-specific configuration files in the form "diecast.env.yml"
	ErrorsPath          string                    `yaml:"errors"                  json:"errors"`                  // The path to the errors template directory
	ExposeEnvVars       []string                  `yaml:"exposeEnvVars"           json:"exposeEnvVars"`           // a list of glob patterns matching environment variable names that should always be exposed
	FaviconPath         string                    `yaml:"favicon"                 json:"favicon"`                 // TODO: favicon autogenerator: Specifies the relative path to the file containing the /favicon.ico file.  This path can point to a Windows Icon (.ico), GIF, PNG, JPEG, or Bitmap (.bmp).  If necessary, the file will be converted and stored in memory to the ICO format.
	FilterEnvVars       []string                  `yaml:"filterEnvVars"           json:"filterEnvVars"`           // a list of glob patterns matching environment variable names that should not be exposed
	GlobalHeaders       map[string]interface{}    `yaml:"globalHeaders,omitempty" json:"globalHeaders,omitempty"` // A set of HTTP headers that should be added to EVERY response Diecast returns, regardless of whether it originates from a template, mount, or other configuration.
	IndexFile           string                    `yaml:"indexFile"               json:"indexFile"`               // The name of the template file to use when a directory is requested.
	LayoutPath          string                    `yaml:"layouts"                 json:"layouts"`                 // The path to the layouts template directory
	Locale              string                    `yaml:"locale"                  json:"locale"`                  // Specify the default locale for pages being served.
	MountConfigs        []MountConfig             `yaml:"mounts"                  json:"mounts"`                  // A list of mount configurations read from the diecast.yml config file.
	Mounts              []Mount                   `yaml:"-"                       json:"-"`                       // The set of all registered mounts.
	OnAddHandler        AddHandlerFunc            `yaml:"-"                       json:"-"`                       // A function that can be used to intercept handlers being added to the server.
	OverridePageObject  map[string]interface{}    `yaml:"-"                       json:"-"`                       //
	PrestartCommands    []*StartCommand           `yaml:"prestart"                json:"prestart"`                // A command that will be executed before the server is started.
	Protocols           map[string]ProtocolConfig `yaml:"protocols"               json:"protocols"`               // Setup global configuration details for Binding Protocols
	RendererMappings    map[string]string         `yaml:"rendererMapping"         json:"rendererMapping"`         // Map file extensions to preferred renderers for a given file type.
	RootPath            string                    `yaml:"root"                    json:"root"`                    // The filesystem location where templates and files are served from
	RoutePrefix         string                    `yaml:"routePrefix"             json:"routePrefix"`             // If specified, all requests must be prefixed with this string.
	StartCommands       []*StartCommand           `yaml:"start"                   json:"start"`                   // A command that will be executed after the server is confirmed running.
	TLS                 *TlsConfig                `yaml:"tls"                     json:"tls"`                     // where SSL/TLS configuration is stored
	TemplatePatterns    []string                  `yaml:"patterns"                json:"patterns"`                // A set of glob patterns specifying which files will be rendered as templates.
	Translations        map[string]interface{}    `yaml:"translations,omitempty"  json:"translations,omitempty"`  // Stores translations for use with the i18n and l10n functions.  Keys values represent the
	TrustedRootPEMs     []string                  `yaml:"trustedRootPEMs"         json:"trustedRootPEMs"`         // List of filenames containing PEM-encoded X.509 TLS certificates that represent trusted authorities.  Use to validate certificates signed by an internal, non-public authority.
	TryExtensions       []string                  `yaml:"tryExtensions"           json:"tryExtensions"`           // Try these file extensions when looking for default (i.e.: "index") files.  If IndexFile has an extension, it will be stripped first.
	TryLocalFirst       bool                      `yaml:"localFirst"              json:"localFirst"`              // Whether to attempt to locate a local file matching the requested path before attempting to find a template.
	VerifyFile          string                    `yaml:"verifyFile"              json:"verifyFile"`              // A file that must exist and be readable before starting the server.
	PreserveConnections bool                      `yaml:"preserveConnections"     json:"preserveConnections"`     // Don't add the "Connection: close" header to every response.
	CSRF                *CSRF                     `yaml:"csrf"                    json:"csrf"`                    // configures CSRF protection
	Log                 LogConfig                 `yaml:"log"                     json:"log"`                     // configure logging
	BeforeHandlers      []Middleware              `yaml:"-"                       json:"-"`                       // contains a stack of Middleware functions that are run before handling the request
	AfterHandlers       []http.HandlerFunc        `yaml:"-"                       json:"-"`                       // contains a stack of HandlerFuncs that are run after handling the request.  These functions cannot stop the request, as it's already been written to the client.
	Protocol            string                    `yaml:"protocol"                json:"protocol"`                // Specify which HTTP protocol to use ("http", "http2", "quic", "http3")
	altRootCaPool       *x509.CertPool
	faviconImageIco     []byte
	fs                  http.FileSystem
	hasUserRoutes       bool
	initialized         bool
	precmd              *exec.Cmd
	mux                 *http.ServeMux
	userRouter          *vestigo.Router
	logwriter           io.Writer
	isTerminalOutput    bool
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
		FilterEnvVars:      DefaultFilterEnvVars,
		GlobalHeaders:      make(map[string]interface{}),
		Protocol:           DefaultProtocol,
		Log: LogConfig{
			Format:      logFormats[`common`],
			Destination: `-`,
			Colorize:    true,
		},
		mux:        http.NewServeMux(),
		userRouter: vestigo.NewRouter(),
	}

	if str, ok := root.(string); ok {
		server.RootPath = str
	} else if fs, ok := root.(http.FileSystem); ok {
		server.SetFileSystem(fs)
	}

	server.mux.HandleFunc(server.rp()+`/`, server.handleRequest)

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
				data = []byte(stringutil.ExpandEnv(string(data)))

				if err := yaml.UnmarshalStrict(data, self); err == nil {
					// apply environment-specific overrides
					if self.Environment != `` {
						eDir, eFile := filepath.Split(filename)
						base := strings.TrimSuffix(eFile, filepath.Ext(eFile))
						ext := filepath.Ext(eFile)
						eFile = fmt.Sprintf("%s.%s%s", base, self.Environment, ext)
						envPath := filepath.Join(eDir, eFile)

						if fileutil.IsNonemptyFile(envPath) {
							if err := self.LoadConfig(envPath); err != nil {
								return fmt.Errorf("failed to load %s: %v", eFile, err)
							}
						}
					}

					// process mount configs into mount instances
					for i, config := range self.MountConfigs {
						if mount, err := NewMountFromSpec(fmt.Sprintf("%s:%s", config.Mount, config.To)); err == nil {
							mountOverwriteIndex := -1

							for i, existing := range self.Mounts {
								if IsSameMount(mount, existing) {
									mountOverwriteIndex = i
									break
								}
							}

							if err := maputil.TaggedStructFromMap(config.Options, mount, `json`); err != nil {
								return fmt.Errorf("mount %d options: %v", i, err)
							}

							if mountOverwriteIndex >= 0 {
								log.Debugf("mount: overwriting mountpoint with new configuration: %v", mount)
								self.Mounts[mountOverwriteIndex] = mount
							} else {
								self.Mounts = append(self.Mounts, mount)
							}
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

func (self *Server) prestart() error {
	if !self.initialized {
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

	return nil
}

// Perform an end-to-end render of a single path, writing the output to the given writer,
// then exit.
func (self *Server) RenderPath(w io.Writer, path string) error {
	path = `/` + strings.TrimPrefix(path, `/`)
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	self.ServeHTTP(rw, req)

	if !rw.Flushed {
		rw.Flush()
	}

	if res := rw.Result(); res.StatusCode < 400 {
		_, err := io.Copy(w, res.Body)
		return err
	} else {
		errbody, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("render failed: %v", sliceutil.Or(string(errbody), res.Status))
	}
}

// Start a long-running webserver.  If provided, the functions provided will be run in parallel
// after the server has started.  If any of them return a non-nil error, the server will stop and
// this method will return that error.
func (self *Server) Serve(workers ...ServeFunc) error {
	var serveable Serveable
	var useTLS bool
	var useUDP bool
	var useSocket string

	// fire off some goroutines for the prestart and start commands (if configured)
	if err := self.prestart(); err != nil {
		return err
	}

	srv := &http.Server{
		Handler: self,
	}

	// work out if we're starting a UNIX socket server
	if addr := self.Address; strings.HasPrefix(addr, `unix:`) {
		useSocket = strings.TrimPrefix(addr, `unix:`)
	} else {
		srv.Addr = addr
	}

	// setup TLSConfig
	if ssl := self.TLS; ssl != nil && ssl.Enable {
		tc := new(tls.Config)

		ssl.CertFile = fileutil.MustExpandUser(ssl.CertFile)
		ssl.KeyFile = fileutil.MustExpandUser(ssl.KeyFile)

		if !fileutil.IsNonemptyFile(ssl.CertFile) {
			return fmt.Errorf("ssl: cert file %q is not readable or does not exist", ssl.CertFile)
		}

		if !fileutil.IsNonemptyFile(ssl.KeyFile) {
			return fmt.Errorf("ssl: key file %q is not readable or does not exist", ssl.KeyFile)
		}

		if mode := ssl.ClientCertMode; mode != `` {
			switch mode {
			case `request`:
				tc.ClientAuth = tls.RequestClientCert
			case `any`:
				tc.ClientAuth = tls.RequireAnyClientCert
			case `verify`:
				tc.ClientAuth = tls.VerifyClientCertIfGiven
			case `require`:
				tc.ClientAuth = tls.RequireAndVerifyClientCert
			default:
				return fmt.Errorf(
					"Invalid value %q for 'ssl_client_certs': must be one of %q, %q, %q, %q.",
					mode,
					`request`,
					`any`,
					`verify`,
					`require`,
				)
			}

			ssl.ClientCAFile = fileutil.MustExpandUser(ssl.ClientCAFile)

			if !fileutil.IsNonemptyFile(ssl.ClientCAFile) {
				return fmt.Errorf("ssl: client CA file %q is not readable or does not exist", ssl.ClientCAFile)
			}

			if pool, err := httputil.LoadCertPool(ssl.ClientCAFile); err == nil {
				tc.ClientCAs = pool
			} else {
				return fmt.Errorf("ssl: client CA: %v", err)
			}
		}

		srv.TLSConfig = tc
		useTLS = true
	}

	// wrap all the various protocol implementations in a common interface
	switch strings.ToLower(self.Protocol) {
	case ``, `http`:
		serveable = srv
	case `http2`:
		h2s := new(http2.Server)

		if useTLS {
			http2.ConfigureServer(srv, h2s)
		} else {
			hnd := srv.Handler
			srv.Handler = h2c.NewHandler(hnd, h2s)
		}

		serveable = srv

	case `quic`, `http3`:
		useUDP = true
		h3s := &http3.Server{
			Server:     srv,
			QuicConfig: nil,
		}

		serveable = &h3serveable{
			Server: h3s,
		}
	default:
		return fmt.Errorf("unknown protocol %q", self.Protocol)
	}

	// take a wildly different path if we're listening on a unix socket
	if useSocket != `` {
		if x, err := fileutil.ExpandUser(useSocket); err == nil {
			useSocket = x
		} else {
			return err
		}

		network := `unix`

		if useUDP {
			network = `unixpacket`
		}

		if _, err := os.Stat(useSocket); err == nil {
			if err := os.Remove(useSocket); err != nil {
				return err
			}
		}

		if listener, err := net.Listen(network, useSocket); err == nil {
			if useTLS {
				return serveable.ServeTLS(listener, self.TLS.CertFile, self.TLS.KeyFile)
			} else {
				return serveable.Serve(listener)
			}
		} else {
			return fmt.Errorf("bad socket: %v", err)
		}
	} else {
		var errchan = make(chan error)

		go func() {
			if useTLS {
				errchan <- serveable.ListenAndServeTLS(self.TLS.CertFile, self.TLS.KeyFile)
			} else {
				errchan <- serveable.ListenAndServe()
			}
		}()

		if len(workers) > 0 {
			// add a slight delay (until the listen port is confirmed open or 1 second, whichever comes first)
			// this is for the benefit of worker functions, so we only do it if there are any
			netutil.WaitForOpen(`tcp`, self.Address, time.Second)

			for _, worker := range workers {
				go func(w ServeFunc) {
					errchan <- w(self)
				}(worker)
			}
		}

		return <-errchan
	}
}

func (self *Server) ListenAndServe(address string) error {
	self.Address = address
	return self.Serve()
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// make sure we close the body no matter what
	if req.Body != nil {
		defer req.Body.Close()
	}

	// set Connection header
	if !self.PreserveConnections {
		w.Header().Set(`Connection`, `close`)
	}

	// initialize if necessary. an error here is severe and panics
	if !self.initialized {
		if err := self.Initialize(); err != nil {
			panic(err.Error())
		}
	}

	// setup a ResponseWriter interceptor that catches status code and bytes written
	// but passes through the Body without buffering it (like httptest.ResponseRecorder does)
	interceptor := intercept(w)
	httputil.RequestSetValue(req, ContextResponseKey, interceptor)

	// process the before stack
	for i, before := range self.BeforeHandlers {
		if proceed := before(interceptor, req); !proceed {
			log.Debugf(
				"[%s] processing halted by middleware %d (msg: %v)",
				reqid(req),
				i,
				httputil.RequestGetValue(req, ContextErrorKey),
			)

			self.respondError(interceptor, req, fmt.Errorf("Middleware halted request"), http.StatusInternalServerError)
			return
		}
	}

	// finally, pass the request on to the ServeMux router
	self.mux.ServeHTTP(interceptor, req)

	// process the middlewares
	for _, after := range self.AfterHandlers {
		after(interceptor, req)
	}
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

	earlyData := self.requestToEvalData(req, header)

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
					layoutName = MustEvalInline(layoutName, nil, earlyFuncs)

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
	finalHeader.Locale = MustEvalInline(finalHeader.Locale, earlyData, earlyFuncs)

	if funcs, data, err := self.GetTemplateData(req, &finalHeader); err == nil {
		start := time.Now()

		// switches allow the template processing to be hijacked/redirected mid-evaluation
		// based on data already evaluated
		if len(finalHeader.Switch) > 0 {
		SwitchCaseLoop:
			for i, swcase := range finalHeader.Switch {
				if swcase == nil {
					continue SwitchCaseLoop
				}

				if swcase.UsePath != `` {
					// if a condition is specified, it must evaluate to a truthy value to proceed
					cond := MustEvalInline(swcase.Condition, data, funcs)
					checkType, checkTypeArg := stringutil.SplitPair(swcase.CheckType, `:`)

					switch checkType {
					case `querystring`, `qs`:
						if checkTypeArg != `` {
							if cond == `` {
								if httputil.Q(req, checkTypeArg) == `` {
									continue SwitchCaseLoop
								}
							} else if httputil.Q(req, checkTypeArg) != cond {
								continue SwitchCaseLoop
							}
						} else {
							return fmt.Errorf("switch checktype %q must specify an argument; e.g.: %q", `querystring`, `querystring:id`)
						}

					case `header`:
						if checkTypeArg != `` {
							if cond == `` {
								if req.Header.Get(checkTypeArg) == `` {
									continue SwitchCaseLoop
								}
							} else if req.Header.Get(checkTypeArg) != cond {
								continue SwitchCaseLoop
							}
						} else {
							return fmt.Errorf("switch checktype %q must specify an argument; e.g.: %q", `header`, `header:X-My-Header`)
						}

					case `expression`, ``:
						if !typeutil.V(cond).Bool() {
							continue SwitchCaseLoop
						}
					default:
						return fmt.Errorf("unknown switch checktype %q", swcase.CheckType)
					}

					if swTemplate, err := self.fs.Open(swcase.UsePath); err == nil {
						if swHeader, swData, err := SplitTemplateHeaderContent(swTemplate); err == nil {
							finalHeader.Switch[i] = nil

							if fh, err := finalHeader.Merge(swHeader); err == nil {
								log.Debugf("[%s] Switch case %d matched, switching to template %v", reqid(req), i, swcase.UsePath)

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
		finalHeader.Renderer = MustEvalInline(finalHeader.Renderer, data, funcs)

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
		log.Debugf("[%s] Performing 307 Temporary Redirect to %v due to binding response handler.", reqid(req), redir)
		writeRequestTimerHeaders(self, w, req)
		http.Redirect(w, req, redir.Error(), http.StatusTemporaryRedirect)
		return nil
	} else {
		return err
	}
}

// Retrieves the set of standard template functions, as well as functions for working
// with data in the current request.
func (self *Server) GetTemplateFunctions(data map[string]interface{}, header *TemplateHeader) FuncMap {
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
		if data, err := readFromFS(self.fs, filename); err == nil {
			return string(data), nil
		} else {
			return ``, err
		}
	}

	// read a file from the serving path and parse it as a template, returning the output.
	funcs[`render`] = func(filename string, overrides ...map[string]interface{}) (string, error) {
		if tpl, err := readFromFS(self.fs, filename); err == nil {
			d := data

			if len(overrides) > 0 && overrides[0] != nil {
				d = overrides[0]
			}

			return EvalInline(string(tpl), d, funcs)
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

// gets a FuncMap and data usable in templates and error pages alike, before bindings are evaluated.
func (self *Server) getPreBindingData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]interface{}) {
	data := self.requestToEvalData(req, header)

	data[`vars`] = make(map[string]interface{})

	publicMountDetails := make([]map[string]interface{}, 0)

	for _, mount := range self.MountConfigs {
		publicMountDetails = append(publicMountDetails, map[string]interface{}{
			`from`: mount.Mount,
			`to`:   mount.To,
		})
	}

	data[`diecast`] = map[string]interface{}{
		`binding_prefix`:    self.BindingPrefix,
		`route_prefix`:      self.rp(),
		`template_patterns`: self.TemplatePatterns,
		`try_local_first`:   self.TryLocalFirst,
		`index_file`:        self.IndexFile,
		`verify_file`:       self.VerifyFile,
		`mounts`:            publicMountDetails,
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
					value = MustEvalInline(value.(string), data, funcs)
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

	return funcs, data
}

func (self *Server) GetTemplateData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]interface{}, error) {
	funcs, data := self.getPreBindingData(req, header)

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

		if pgConfig := binding.Paginate; pgConfig != nil {
			results := make([]map[string]interface{}, 0)
			proceed := true

			var total int64
			var count int64
			var soFar int64

			page := 1

			lastPage := maputil.M(&ResultsPage{
				Page:    page,
				Counter: soFar,
			}).MapNative(`json`)

			for proceed {
				suffix := fmt.Sprintf("binding(%s):page(%d)", binding.Name, page+1)

				bindings[binding.Name] = binding.Fallback
				data[`page`] = lastPage

				if len(binding.Params) == 0 {
					binding.Params = make(map[string]interface{})
				}

				if len(binding.Headers) == 0 {
					binding.Headers = make(map[string]string)
				}

				// eval the URL
				binding.Resource = MustEvalInline(binding.Resource, data, funcs, suffix)

				// eval / set querystring params
				for qsk, qsv := range pgConfig.QueryStrings {
					binding.Params[qsk] = typeutil.Auto(MustEvalInline(qsv, data, funcs, suffix))
				}

				// eval / set request headers
				for hk, hv := range pgConfig.Headers {
					binding.Headers[hk] = MustEvalInline(hv, data, funcs, suffix)
				}

				v, err := binding.Evaluate(req, header, data, funcs)

				if err == nil {
					asMap := maputil.M(v)

					total = typeutil.Int(MustEvalInline(pgConfig.Total, asMap.MapNative(), funcs, suffix))
					count = typeutil.Int(MustEvalInline(pgConfig.Count, asMap.MapNative(), funcs, suffix))
					soFar += count

					log.Debugf("[%v] paginated binding %q: total=%v count=%v soFar=%v", reqid(req), binding.Name, total, count, soFar)

					proceed = !typeutil.Bool(MustEvalInline(pgConfig.Done, asMap.MapNative(), funcs, suffix))

					if pgConfig.Maximum > 0 && soFar >= pgConfig.Maximum {
						proceed = false
					}

					if !proceed {
						log.Debugf("[%v] paginated binding %q: proceed is false, this is the last loop", reqid(req), binding.Name)
					}

					thisPage := maputil.M(&ResultsPage{
						Total:   total,
						Page:    page,
						Last:    !proceed,
						Counter: soFar,
						Range: []int64{
							(soFar - count),
							soFar,
						},
					}).MapNative(`json`)

					if output, err := ApplyJPath(v, pgConfig.Data); err == nil {
						v = output
					} else {
						return funcs, nil, err
					}

					thisPage[`data`] = v
					results = append(results, thisPage)
					data[`page`] = maputil.M(thisPage).MapNative(`json`)
					lastPage = thisPage

					bindings[binding.Name] = results
					data[`bindings`] = bindings
				} else if redir, ok := err.(RedirectTo); ok {
					return funcs, nil, redir
				} else {
					if err != ErrSkipEval {
						log.Warningf("[%s] Binding %q (iteration %d) failed: %v", reqid(req), binding.Name, i, err)
					}

					if binding.OnError == ActionContinue {
						continue
					} else if binding.OnError == ActionBreak {
						break
					} else if !binding.Optional {
						return funcs, nil, err
					}
				}

				data[`bindings`] = bindings
				page++
			}

			bindings[binding.Name] = results
			data[`bindings`] = bindings

		} else if binding.Repeat == `` {
			bindings[binding.Name] = binding.Fallback
			data[`bindings`] = bindings

			v, err := binding.Evaluate(req, header, data, funcs)

			if err == nil && v != nil {
				bindings[binding.Name] = v
				data[`bindings`] = bindings
			} else if redir, ok := err.(RedirectTo); ok {
				return funcs, nil, redir
			} else if v == nil && binding.Fallback != nil {
				bindings[binding.Name] = binding.Fallback
			} else {
				if err != ErrSkipEval {
					log.Warningf("[%s] Binding %q failed: %v", reqid(req), binding.Name, err)
				}

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
					MustEvalInline(repeatExpr, data, funcs),
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
				flags[name] = typeutil.V(MustEvalInline(fmt.Sprintf("%v", def), data, funcs)).Bool()
			}
		}

		data[`flags`] = flags
	}

	return funcs, data, nil
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
				if mimetype, err := figureOutMimeType(stat.Name(), file); err == nil {
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
			log.Debugf("[%s] process mounts: buffered %d bytes from request body", reqid(req), len(data))
		}

		body = bytes.NewReader(data)
		req.Body = ioutil.NopCloser(body)
	} else {
		return nil, nil, err
	}

	var lastErr error

	// find a mount that has this file
	for _, mount := range self.Mounts {
		// seek the body buffer back to the beginning
		if _, err := body.Seek(0, 0); err != nil {
			return nil, nil, err
		}

		if mount.WillRespondTo(requestPath, req, body) {
			// attempt to open the file entry
			mountResponse, err := mount.OpenWithType(requestPath, req, body)
			lastErr = err

			if err == nil {
				log.Debugf("mount %v handled %q", mount.GetMountPoint(), requestPath)
				return mount, mountResponse, nil
			} else if IsHardStop(err) {
				return nil, nil, err
			}
		}
	}

	if _, err := body.Seek(0, 0); err != nil {
		return nil, nil, err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("%q not found", requestPath)
	}

	return nil, nil, lastErr
}

func (self *Server) respondError(w http.ResponseWriter, req *http.Request, resErr error, code int) {
	tmpl := NewTemplate(`error`, HtmlEngine)

	if resErr == nil {
		resErr = fmt.Errorf("Unknown Error")
	}

	if c := httputil.RequestGetValue(req, ContextStatusKey).Int(); c > 0 {
		code = int(c)
	}

	for _, filename := range []string{
		fmt.Sprintf("%s/%d.html", self.ErrorsPath, code),
		fmt.Sprintf("%s/%dxx.html", self.ErrorsPath, int(code/100.0)),
		fmt.Sprintf("%s/default.html", self.ErrorsPath),
	} {
		if f, err := self.fs.Open(filename); err == nil {
			funcs, errorData := self.getPreBindingData(req, self.BaseHeader)
			if msg := httputil.RequestGetValue(req, ContextErrorKey).String(); msg != `` {
				errorData[`error`] = msg
			} else {
				errorData[`error`] = resErr.Error()
			}

			errorData[`errorcode`] = code
			tmpl.Funcs(funcs)

			if err := tmpl.ParseFrom(f); err == nil {
				w.Header().Set(`Content-Type`, fileutil.GetMimeType(filename, `text/html; charset=utf-8`))

				if err := tmpl.renderWithRequest(req, w, errorData, ``); err == nil {
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
				header := TemplateHeader{
					QueryJoiner: DefaultQueryJoiner,
				}

				if parts[1] != nil {
					header.lines = len(strings.Split(string(parts[1]), "\n"))

					if err := yaml.UnmarshalStrict(parts[1], &header); err != nil {
						return nil, nil, err
					}
				}

				parts[2] = bytes.TrimLeft(parts[2], "\r\n")

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

func csrftoken(req *http.Request) string {
	return httputil.RequestGetValue(req, ContextCsrfToken).String()
}

func reqid(req *http.Request) string {
	return httputil.RequestGetValue(req, ContextRequestKey).String()
}

func reqres(req *http.Request) *statusInterceptor {
	if w := httputil.RequestGetValue(req, ContextResponseKey).Value; w != nil {
		if rw, ok := w.(*statusInterceptor); ok {
			return rw
		}
	}

	panic("no ResponseWriter for request")
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

func (self *Server) requestToEvalData(req *http.Request, header *TemplateHeader) map[string]interface{} {
	var rv = make(map[string]interface{})
	var request = RequestInfo{
		Headers: make(map[string]interface{}),
		URL: RequestUrlInfo{
			Query:  make(map[string]interface{}),
			Params: make(map[string]interface{}),
		},
	}

	var qj = DefaultQueryJoiner
	var hj = DefaultHeaderJoiner

	// query strings
	// ------------------------------------------------------------------------
	if header != nil {
		for dK, dV := range header.Defaults {
			request.URL.Query[dK] = stringutil.Autotype(dV)
		}

		if header.QueryJoiner != `` {
			qj = header.QueryJoiner
		}

		if header.HeaderJoiner != `` {
			hj = header.HeaderJoiner
		}
	}

	for k, v := range req.URL.Query() {
		if vv := strings.Join(v, qj); !typeutil.IsZero(vv) {
			request.URL.Query[k] = stringutil.Autotype(vv)
		}
	}

	// response headers
	// ------------------------------------------------------------------------
	if header != nil {
		for dK, dV := range header.DefaultHeaders {
			dK = stringutil.Underscore(strings.ToLower(dK))
			request.Headers[dK] = stringutil.Autotype(dV)
		}
	}

	for k, v := range req.Header {
		if vv := strings.Join(v, hj); !typeutil.IsZero(vv) {
			k = stringutil.Underscore(strings.ToLower(k))
			request.Headers[k] = stringutil.Autotype(vv)
		}
	}

	request.ID = reqid(req)
	request.Timestamp = time.Now().UnixNano()
	request.Method = req.Method
	request.Protocol = req.Proto
	request.ContentLength = req.ContentLength

	if te := req.TransferEncoding; te == nil {
		request.TransferEncoding = []string{`identity`}
	} else {
		request.TransferEncoding = te
	}

	addr, port, _ := net.SplitHostPort(req.RemoteAddr)

	request.RemoteIP = addr
	request.RemotePort = int(typeutil.Int(port))
	request.RemoteAddr = req.RemoteAddr

	host, port, _ := net.SplitHostPort(sliceutil.OrString(req.URL.Host, req.Host))

	request.Host = host

	request.URL.Unmodified = req.RequestURI
	request.URL.String = req.URL.String()
	request.URL.Scheme = req.URL.Scheme
	request.URL.Host = host
	request.URL.Port = int(typeutil.Int(port))
	request.URL.Path = req.URL.Path
	request.URL.Fragment = req.URL.Fragment

	if header != nil {
		request.URL.Params = header.UrlParams
	}

	if state := req.TLS; state != nil {
		request.TLS = new(RequestTlsInfo)

		sslclients := make([]RequestTlsCertInfo, 0)

		for _, pcrt := range state.PeerCertificates {
			sslclients = append(sslclients, RequestTlsCertInfo{
				Issuer:         pkixNameToMap(pcrt.Issuer),
				Subject:        pkixNameToMap(pcrt.Subject),
				NotBefore:      pcrt.NotBefore,
				NotAfter:       pcrt.NotAfter,
				SecondsLeft:    int(-1 * time.Since(pcrt.NotAfter).Round(time.Second).Seconds()),
				OcspServer:     pcrt.OCSPServer,
				IssuingCertUrl: pcrt.IssuingCertificateURL,
				Version:        pcrt.Version,
				SerialNumber:   pcrt.SerialNumber.String(),
				SubjectAlternativeName: &RequestTlsCertSan{
					DNSNames:       pcrt.DNSNames,
					EmailAddresses: pcrt.EmailAddresses,
					IPAddresses:    sliceutil.Stringify(pcrt.IPAddresses),
					URIs:           sliceutil.Stringify(pcrt.URIs),
				},
			})
		}

		request.TLS.Version = tlstext.Version(state.Version)
		request.TLS.HandshakeComplete = state.HandshakeComplete
		request.TLS.DidResume = state.DidResume
		request.TLS.CipherSuite = tlstext.CipherSuite(state.CipherSuite)
		request.TLS.NegotiatedProtocol = state.NegotiatedProtocol
		request.TLS.NegotiatedProtocolIsMutual = state.NegotiatedProtocolIsMutual
		request.TLS.ServerName = state.ServerName
		request.TLS.TlsUnique = state.TLSUnique

		if len(sslclients) > 0 {
			request.TLS.Client = sslclients[0]
			request.TLS.ClientChain = sslclients[1:]
		}

		if request.URL.Scheme == `` {
			request.URL.Scheme = `https`
		}
	} else if request.URL.Scheme == `` {
		request.URL.Scheme = `http`
	}

	request.CSRFToken = csrftoken(req)
	rv[`request`] = maputil.M(request).MapNative(`json`)

	// environment variables
	env := make(map[string]interface{})

	for _, pair := range os.Environ() {
		key, value := stringutil.SplitPair(pair, `=`)
		key = envKeyNorm(key)

		if self.mayExposeEnvVar(key) {
			env[key] = stringutil.Autotype(value)
		}
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

func (self *Server) mayExposeEnvVar(name string) bool {
	name = envKeyNorm(name)

	for _, f := range self.ExposeEnvVars {
		if glob.MustCompile(envKeyNorm(f)).Match(name) {
			return true
		}
	}

	for _, f := range self.FilterEnvVars {
		if glob.MustCompile(envKeyNorm(f)).Match(name) {
			return false
		}
	}

	return true
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

// called by the cleanup middleware to log the completed request according to LogFormat.
func (self *Server) logreq(w http.ResponseWriter, req *http.Request) {
	if tm := getRequestTimer(req); tm != nil {
		format := logFormats[self.Log.Format]

		if format == `` {
			if self.Log.Format != `` {
				format = self.Log.Format
			} else {
				return
			}
		}

		if self.logwriter == nil {
			// discard by default, unless some brave configuration below changes this
			self.logwriter = ioutil.Discard

			switch lf := strings.ToLower(self.Log.Destination); lf {
			case ``, `none`, `false`:
				return
			case `-`, `stdout`:
				self.isTerminalOutput = true
				self.logwriter = os.Stdout
			case `stderr`:
				self.isTerminalOutput = true
				self.logwriter = os.Stderr
			case `syslog`:
				log.Warningf("logfile: %q destination is not implemented", lf)
				return
			default:
				if self.Log.Truncate {
					os.Truncate(self.Log.Destination, 0)
				}

				if f, err := os.OpenFile(self.Log.Destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					self.logwriter = f
				} else {
					log.Warningf("logfile: failed to open logfile: %v", err)
					return
				}
			}
		}

		interceptor := reqres(req)
		rh, rp := stringutil.SplitPair(req.RemoteAddr, `:`)
		code := typeutil.String(interceptor.code)

		if self.isTerminalOutput && self.Log.Colorize {
			if interceptor.code < 300 {
				code = log.CSprintf("${green}%d${reset}", interceptor.code)
			} else if interceptor.code < 400 {
				code = log.CSprintf("${cyan}%d${reset}", interceptor.code)
			} else if interceptor.code < 500 {
				code = log.CSprintf("${yellow}%d${reset}", interceptor.code)
			} else {
				code = log.CSprintf("${red}%d${reset}", interceptor.code)
			}
		}

		logContext := maputil.M(map[string]interface{}{
			`host`:                req.Host,
			`method`:              req.Method,
			`protocol_major`:      req.ProtoMajor,
			`protocol_minor`:      req.ProtoMinor,
			`protocol`:            req.Proto,
			`remote_address`:      rh,
			`remote_address_port`: typeutil.Int(rp),
			`request_id`:          reqid(req),
			`request_length`:      req.ContentLength,
			`request_started_at`:  tm.StartedAt,
			`duration`:            httputil.RequestGetValue(req, `duration`).Duration(),
			`response_length`:     interceptor.bytesWritten,
			`scheme`:              req.URL.Scheme,
			`status_code`:         code,
			`status_text`:         http.StatusText(interceptor.code),
			`url_hostname`:        req.URL.Hostname(),
			`url_port`:            typeutil.Int(req.URL.Port()),
			`url`:                 req.URL.String(),
		})

		logContext.Fprintf(self.logwriter, format)
	} else {
		bugWarning()
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

func pkixNameToMap(name pkix.Name) (certname RequestTlsCertName) {
	if len(name.Country) > 0 {
		certname.Country = strings.Join(name.Country, `,`)
	}

	if len(name.Organization) > 0 {
		certname.Organization = strings.Join(name.Organization, `,`)
	}

	if len(name.OrganizationalUnit) > 0 {
		certname.OrganizationalUnit = strings.Join(name.OrganizationalUnit, `,`)
	}

	if len(name.Locality) > 0 {
		certname.Locality = strings.Join(name.Locality, `,`)
	}

	if len(name.Province) > 0 {
		certname.State = strings.Join(name.Province, `,`)
	}

	if len(name.StreetAddress) > 0 {
		certname.StreetAddress = strings.Join(name.StreetAddress, `,`)
	}

	if len(name.PostalCode) > 0 {
		certname.PostalCode = strings.Join(name.PostalCode, `,`)
	}

	certname.SerialNumber = name.SerialNumber
	certname.CommonName = name.CommonName
	return
}

func envKeyNorm(in string) string {
	in = strings.ToLower(in)

	return in
}

func formatRequest(req *http.Request) string {
	var request []string // Add the request string

	url := fmt.Sprintf("%s %v %v", req.Method, req.URL, req.Proto)

	request = append(request, url)
	request = append(request, fmt.Sprintf("host: %s", req.Host))
	headerNames := maputil.StringKeys(req.Header)
	sort.Strings(headerNames)

	for _, name := range headerNames {
		headers := req.Header[name]
		name = strings.ToLower(name)

		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	data, err := ioutil.ReadAll(req.Body)
	req.Body.Close()

	if err == nil {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(data))
		return strings.Join(request, "\r\n") + "\r\n\r\n" + string(data)
	} else {
		request = append(request, fmt.Sprintf("\nFAILED to read body: %v", err))
	}

	return strings.Join(request, "\n")
}
