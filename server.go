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
	"regexp"
	"sort"
	"strings"
	"sync"
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
	"github.com/ghetzel/ratelimit"
	"github.com/gobwas/glob"
	jwt "github.com/golang-jwt/jwt"
	"github.com/husobee/vestigo"
	"github.com/mattn/go-shellwords"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/signalsciences/tlstext"
	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
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

const weirdPathsInHostnamesPlaceholder = "\u2044"

func init() {
	maputil.UnmarshalStructTag = `json`
	stringutil.ExpandEnvPreserveIfEmpty = true
}

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
const JaegerSpanKey = `jaeger-span`
const RequestBodyKey = `request-body`

var HeaderSeparator = []byte{'-', '-', '-'}
var DefaultIndexFile = `index.html`
var DefaultVerifyFile = `/` + DefaultIndexFile
var DefaultTemplatePatterns = []string{`*.html`, `*.md`, `*.scss`}
var DefaultAutocompressPatterns = []string{`*.zip`, `*.docx`, `*.xlsx`, `*.pptx`}
var DefaultTryExtensions = []string{`html`, `md`}
var DefaultAutoindexFilename = `/autoindex.html`
var DefaultRequestBodyPreload int64 = 1048576

var DefaultAutolayoutPatterns = []string{
	`*.html`,
	`*.md`,
}

var DefaultRendererMappings = map[string]string{
	`md`:   `markdown`,
	`scss`: `sass`,
	`pptx`: `ooxml`,
	`xlsx`: `ooxml`,
	`docx`: `ooxml`,
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
	Format      string `yaml:"format"               json:"format"`      // configure the output format for logging requests
	Destination string `yaml:"destination"          json:"destination"` // specify where logs should be written to
	Truncate    bool   `yaml:"truncate"             json:"truncate"`    // if true, the output log file will be truncated on startup
	Colorize    bool   `yaml:"colorize"             json:"colorize"`    // if false, log output will not be colorized
}

type RateLimitConfig struct {
	Enable    bool   `yaml:"enable"     json:"enable"`
	Limit     string `yaml:"limit"      json:"limit"`      // Specify a rate limit string (e.g.: "1r/s", "200r/m")
	PerClient bool   `yaml:"per_client" json:"per_client"` // Specify that the limit should be applied per-client instead of globally.
	Penalty   string `yaml:"penalty"    json:"penalty"`    // An amount of time to sleep instead of returning an HTTP 429 error on rate limited requests
}

func (self *RateLimitConfig) KeyFor(req *http.Request) string {
	if self.PerClient {
		return req.RemoteAddr
	} else {
		return `__global__`
	}
}

type TraceMapping struct {
	Match   string `yaml:"match"   json:"match"`   // A regular expression used to match candidate trace operation names
	Replace string `yaml:"replace" json:"replace"` // A string that will replace matching operation names.
	rx      *regexp.Regexp
}

func (self *TraceMapping) TraceName(candidate string) (string, bool) {
	if self.Match != `` {
		if self.rx == nil {
			if rx, err := regexp.Compile(self.Match); err == nil {
				self.rx = rx
			}
		}
	}

	if self.rx != nil {
		if self.rx.MatchString(candidate) {
			return self.rx.ReplaceAllString(candidate, self.Replace), true
		}
	}

	return ``, false
}

type JWTConfig struct {
	Algorithm string                 `yaml:"alg"     json:"alg"`     // The JWT signing algorithm to use (default: HS256)
	Secret    string                 `yaml:"secret"  json:"secret"`  // The JWT secret used to sign payloads
	Claims    map[string]interface{} `yaml:"claims"  json:"claims"`  // The claims being made (i.e.: the payload that will be converted to JSON)
	Expires   interface{}            `yaml:"expires" json:"expires"` // A duration string representing how long issued tokens will be valid for (default: 60s)
	Issuer    string                 `yaml:"issuer"  json:"issuer"`  // The JWT issuer
	Subject   string                 `yaml:"subject" json:"subject"`
}

func (self *JWTConfig) SignedString(input string) (string, error) {
	var alg = typeutil.OrString(self.Algorithm, `HS256`)

	if signer := jwt.GetSigningMethod(alg); signer != nil {
		return signer.Sign(input, []byte(self.Secret))
	} else {
		return ``, fmt.Errorf("invalid signing algorithm %q", alg)
	}
}

func (self *JWTConfig) Issue(tpldata map[string]interface{}, funcs FuncMap) (string, error) {
	var now = time.Now()
	var alg = typeutil.OrString(self.Algorithm, `HS256`)
	var expiry = typeutil.OrDuration(self.Expires, `60s`)

	if signer := jwt.GetSigningMethod(alg); signer != nil {
		var claims jwt.Claims

		if len(self.Claims) == 0 {
			claims = jwt.StandardClaims{
				Id:        stringutil.UUID().String(),
				IssuedAt:  now.Unix(),
				ExpiresAt: now.Add(expiry).Unix(),
				Issuer:    ShouldEvalInline(self.Issuer, tpldata, funcs).String(),
				Subject:   ShouldEvalInline(self.Subject, tpldata, funcs).String(),
				NotBefore: now.Unix(),
			}
		} else {
			var c = jwt.MapClaims(self.Claims)

			if self.Issuer != `` {
				c[`iss`] = self.Issuer
			}

			c[`iat`] = now.Unix()
			c[`exp`] = now.Add(expiry).Unix()
			c[`jti`] = stringutil.UUID().String()

			for k, v := range c {
				c[k] = ShouldEvalInline(v, tpldata, funcs).Value
			}

			claims = c
		}

		// put it all together: sign the JSONified claims using the given signing method and secret
		return jwt.NewWithClaims(signer, claims).SignedString([]byte(self.Secret))
	} else {
		return ``, fmt.Errorf("invalid signing algorithm %q", alg)
	}
}

type JaegerConfig struct {
	Enable                  bool                   `yaml:"enable"                  json:"enable"`                  // Explicitly enable or disable Jaeger tracing
	ServiceName             string                 `yaml:"service"                 json:"service"`                 // Set the service name that traces will fall under.
	Agent                   string                 `yaml:"agent"                   json:"agent"`                   // Specify the host:port of a local UDP agent to send traces to.
	Collector               string                 `yaml:"collector"               json:"collector"`               // Specify the collector address to sent traces to.  Overrides "agent" if set.
	Username                string                 `yaml:"username"                json:"username"`                // Provides a username to authenticate with the collector.
	Password                string                 `yaml:"password"                json:"password"`                // Provides a password to authenticate with the collector.
	QueueSize               int                    `yaml:"queueSize"               json:"queueSize"`               // Specify the size of the queue for outgoing reports.
	FlushInterval           string                 `yaml:"flushInterval"           json:"flushInterval"`           // Duration specifying how frequently queued reports should be flushed.
	Tags                    map[string]interface{} `yaml:"tags"                    json:"tags"`                    // A set of key-value pairs that are included in every trace.
	SamplingType            string                 `yaml:"sampling"                json:"sampling"`                // Specifies the type of sampling to use: const, probabilistic, rateLimiting, or remote.
	SamplingParam           float64                `yaml:"samplingParam"           json:"samplingParam"`           // A type-specific parameter used to configure that type of sampling; const: 0 or 1, probabilistic: 0.0-1.0, rateLimiting: max number of spans per seconds, remote: same as probabilistic.
	SamplingServerURL       string                 `yaml:"samplingUrl"             json:"samplingUrl"`             // The sampling server URL for the "remote" sampling type.
	SamplingRefreshInterval string                 `yaml:"samplingRefreshInterval" json:"samplingRefreshInterval"` // How frequently to poll the remote sampling server.
	SamplingMaxOperations   int                    `yaml:"samplingMaxOps"          json:"samplingMaxOps"`          // A maximum number of operations for certain sampling modes.
	OperationsMappings      []*TraceMapping        `yaml:"operations"              json:"operations"`              // Maps regular expressions used to match specific routes to the operation name that will be emitted in traces. Without a matching expression, traces will be named by the calling HTTP method and Request URI.  The string being tested by these regular expressions is the one that would be emitted otherwise; so "GET /path/to/file"
}

type Server struct {
	Actions              []*Action                 `yaml:"actions"                 json:"actions"`                 // Configure routes and actions to execute when those routes are requested.
	AdditionalFunctions  template.FuncMap          `yaml:"-"                       json:"-"`                       // Allow for the programmatic addition of extra functions for use in templates.
	Address              string                    `yaml:"address"                 json:"address"`                 // The host:port address the server is listening on
	Authenticators       AuthenticatorConfigs      `yaml:"authenticators"          json:"authenticators"`          // A set of authenticator configurations used to protect some or all routes.
	Autoindex            bool                      `yaml:"autoindex"               json:"autoindex"`               // Specify that requests that terminate at a filesystem directory should automatically generate an index listing of that directory.
	AutoindexTemplate    string                    `yaml:"autoindexTemplate"       json:"autoindexTemplate"`       // If Autoindex is enabled, this allows the template used to generate the index page to be customized.
	AutolayoutPatterns   []string                  `yaml:"autolayoutPatterns"      json:"autolayoutPatterns"`      // Which types of files will automatically have layouts applied.
	BaseHeader           *TemplateHeader           `yaml:"header"                  json:"header"`                  // A default header that all templates will inherit from.
	BinPath              string                    `yaml:"-"                       json:"-"`                       // Exposes the location of the diecast binary
	BindingPrefix        string                    `yaml:"bindingPrefix"           json:"bindingPrefix"`           // Specify a string to prefix all binding resource values that start with "/"
	Bindings             SharedBindingSet          `yaml:"bindings"                json:"bindings"`                // Top-level bindings that apply to every rendered template
	DefaultPageObject    map[string]interface{}    `yaml:"-"                       json:"-"`                       //
	DisableCommands      bool                      `yaml:"disable_commands"        json:"disable_commands"`        // Disable the execution of PrestartCommands and StartCommand .
	DisableTimings       bool                      `yaml:"disableTimings"          json:"disableTimings"`          // Disable emitting per-request Server-Timing headers to aid in tracing bottlenecks and performance issues.
	EnableDebugging      bool                      `yaml:"debug"                   json:"debug"`                   // Enables additional options for debugging applications. Caution: can expose secrets and other sensitive data.
	DebugDumpRequests    map[string]string         `yaml:"debugDumpRequests"       json:"debugDumpRequests"`       // An object keyed on path globs whose values are a directory where matching requests are dumped in their entirety as text files.
	EnableLayouts        bool                      `yaml:"enableLayouts"           json:"enableLayouts"`           // Specifies whether layouts are enabled
	Environment          string                    `yaml:"environment"             json:"environment"`             // Specify the environment for loading environment-specific configuration files in the form "diecast.env.yml"
	ErrorsPath           string                    `yaml:"errors"                  json:"errors"`                  // The path to the errors template directory
	ExposeEnvVars        []string                  `yaml:"exposeEnvVars"           json:"exposeEnvVars"`           // a list of glob patterns matching environment variable names that should always be exposed
	FaviconPath          string                    `yaml:"favicon"                 json:"favicon"`                 // TODO: favicon autogenerator: Specifies the relative path to the file containing the /favicon.ico file.  This path can point to a Windows Icon (.ico), GIF, PNG, JPEG, or Bitmap (.bmp).  If necessary, the file will be converted and stored in memory to the ICO format.
	FilterEnvVars        []string                  `yaml:"filterEnvVars"           json:"filterEnvVars"`           // a list of glob patterns matching environment variable names that should not be exposed
	GlobalHeaders        map[string]interface{}    `yaml:"globalHeaders,omitempty" json:"globalHeaders,omitempty"` // A set of HTTP headers that should be added to EVERY response Diecast returns, regardless of whether it originates from a template, mount, or other configuration.
	IndexFile            string                    `yaml:"indexFile"               json:"indexFile"`               // The name of the template file to use when a directory is requested.
	LayoutPath           string                    `yaml:"layouts"                 json:"layouts"`                 // The path to the layouts template directory
	Locale               string                    `yaml:"locale"                  json:"locale"`                  // Specify the default locale for pages being served.
	MountConfigs         []MountConfig             `yaml:"mounts"                  json:"mounts"`                  // A list of mount configurations read from the diecast.yml config file.
	Mounts               []Mount                   `yaml:"-"                       json:"-"`                       // The set of all registered mounts.
	OnAddHandler         AddHandlerFunc            `yaml:"-"                       json:"-"`                       // A function that can be used to intercept handlers being added to the server.
	OverridePageObject   map[string]interface{}    `yaml:"-"                       json:"-"`                       //
	PrestartCommands     []*StartCommand           `yaml:"prestart"                json:"prestart"`                // A command that will be executed before the server is started.
	Protocols            map[string]ProtocolConfig `yaml:"protocols"               json:"protocols"`               // Setup global configuration details for Binding Protocols
	RendererMappings     map[string]string         `yaml:"rendererMapping"         json:"rendererMapping"`         // Map file extensions to preferred renderers for a given file type.
	RootPath             string                    `yaml:"root"                    json:"root"`                    // The filesystem location where templates and files are served from
	RoutePrefix          string                    `yaml:"routePrefix"             json:"routePrefix"`             // If specified, all requests must be prefixed with this string.
	StartCommands        []*StartCommand           `yaml:"start"                   json:"start"`                   // A command that will be executed after the server is confirmed running.
	TLS                  *TlsConfig                `yaml:"tls"                     json:"tls"`                     // where SSL/TLS configuration is stored
	TemplatePatterns     []string                  `yaml:"patterns"                json:"patterns"`                // A set of glob patterns specifying which files will be rendered as templates.
	Translations         map[string]interface{}    `yaml:"translations,omitempty"  json:"translations,omitempty"`  // Stores translations for use with the i18n and l10n functions.  Keys values represent the
	TrustedRootPEMs      []string                  `yaml:"trustedRootPEMs"         json:"trustedRootPEMs"`         // List of filenames containing PEM-encoded X.509 TLS certificates that represent trusted authorities.  Use to validate certificates signed by an internal, non-public authority.
	TryExtensions        []string                  `yaml:"tryExtensions"           json:"tryExtensions"`           // Try these file extensions when looking for default (i.e.: "index") files.  If IndexFile has an extension, it will be stripped first.
	TryLocalFirst        bool                      `yaml:"localFirst"              json:"localFirst"`              // Whether to attempt to locate a local file matching the requested path before attempting to find a template.
	VerifyFile           string                    `yaml:"verifyFile"              json:"verifyFile"`              // A file that must exist and be readable before starting the server.
	PreserveConnections  bool                      `yaml:"preserveConnections"     json:"preserveConnections"`     // Don't add the "Connection: close" header to every response.
	CSRF                 *CSRF                     `yaml:"csrf"                    json:"csrf"`                    // configures CSRF protection
	Log                  LogConfig                 `yaml:"log"                     json:"log"`                     // configure logging
	BeforeHandlers       []Middleware              `yaml:"-"                       json:"-"`                       // contains a stack of Middleware functions that are run before handling the request
	AfterHandlers        []http.HandlerFunc        `yaml:"-"                       json:"-"`                       // contains a stack of HandlerFuncs that are run after handling the request.  These functions cannot stop the request, as it's already been written to the client.
	Protocol             string                    `yaml:"protocol"                json:"protocol"`                // Specify which HTTP protocol to use ("http", "http2")
	RateLimit            *RateLimitConfig          `yaml:"ratelimit"               json:"ratelimit"`               // Specify a rate limiting configuration.
	BindingTimeout       interface{}               `yaml:"bindingTimeout"          json:"bindingTimeout"`          // Sets the default timeout for bindings that don't explicitly set one.
	JaegerConfig         *JaegerConfig             `yaml:"jaeger"                  json:"jaeger"`                  // Configures distributed tracing using Jaeger.
	AutocompressPatterns []string                  `yaml:"autocompress"            json:"autocompress"`            // A set of glob patterns indicating directories whose contents will be delivered as ZIP files
	RequestBodyPreload   int64                     `yaml:"requestPreload"          json:"requestPreload"`          // Maximum number of bytes to read from a request body for the purpose of automatically parsing it.  Requests larger than this will not be available to templates.
	JWT                  map[string]*JWTConfig     `yaml:"jwt"                     json:"jwt"`                     // Contains configurations for generating JSON Web Tokens in templates.
	altRootCaPool        *x509.CertPool
	faviconImageIco      []byte
	fs                   http.FileSystem
	hasUserRoutes        bool
	initialized          bool
	precmd               *exec.Cmd
	mux                  *http.ServeMux
	userRouter           *vestigo.Router
	logwriter            io.Writer
	isTerminalOutput     bool
	rateLimiter          *ratelimit.Limit
	jaegerCfg            *jaegercfg.Configuration
	opentrace            opentracing.Tracer
	otcloser             io.Closer
	viaConstructor       bool
	sharedBindingData    sync.Map
	lockGetFunctions     sync.Mutex
}

func NewServer(root interface{}, patterns ...string) *Server {
	describeTimer(`tpl`, `Diecast Template Rendering`)

	var server = &Server{
		RootPath:           `.`,
		TemplatePatterns:   patterns,
		Authenticators:     make(AuthenticatorConfigs, 0),
		Bindings:           make(SharedBindingSet, 0),
		DefaultPageObject:  make(map[string]interface{}),
		Mounts:             make([]Mount, 0),
		OverridePageObject: make(map[string]interface{}),
		GlobalHeaders:      make(map[string]interface{}),
		EnableLayouts:      true,
		RequestBodyPreload: DefaultRequestBodyPreload,
		mux:                http.NewServeMux(),
		userRouter:         vestigo.NewRouter(),
		viaConstructor:     true,
	}

	server.populateDefaults()

	if str, ok := root.(string); ok {
		server.RootPath = str
	} else if fs, ok := root.(http.FileSystem); ok {
		server.SetFileSystem(fs)
	}

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
			defer file.Close()
			return self.LoadConfigFromReader(file, filename)
		} else {
			return err
		}
	}

	return nil
}

func (self *Server) LoadConfigFromReader(file io.Reader, filename string) error {
	if data, err := ioutil.ReadAll(file); err == nil && len(data) > 0 {
		data = []byte(stringutil.ExpandEnv(string(data)))

		if err := yaml.UnmarshalStrict(data, self); err == nil {
			// apply environment-specific overrides
			if self.Environment != `` && filename != `` {
				eDir, eFile := filepath.Split(filename)
				var base = strings.TrimSuffix(eFile, filepath.Ext(eFile))
				var ext = filepath.Ext(eFile)
				eFile = fmt.Sprintf("%s.%s%s", base, self.Environment, ext)
				var envPath = filepath.Join(eDir, eFile)

				if fileutil.IsNonemptyFile(envPath) {
					if err := self.LoadConfig(envPath); err != nil {
						return fmt.Errorf("failed to load %s: %v", eFile, err)
					}
				}
			}

			// process mount configs into mount instances
			for i, config := range self.MountConfigs {
				if mount, err := NewMountFromSpec(fmt.Sprintf("%s:%s", config.Mount, config.To)); err == nil {
					var mountOverwriteIndex = -1

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

// Read a file from the underlying root filesystem, satisfying the http.FileSystem interface.
func (self *Server) Open(name string) (http.File, error) {
	if self.fs == nil {
		return nil, fmt.Errorf("no filesystem")
	} else {
		return self.fs.Open(name)
	}
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

func (self *Server) populateDefaults() {
	if self.mux == nil {
		self.mux = http.NewServeMux()
	}

	if self.userRouter == nil {
		self.userRouter = vestigo.NewRouter()
	}

	if self.Log.Format == `` {
		self.Log.Format = logFormats[`common`]
		self.Log.Destination = `-`
		self.Log.Colorize = true
	}

	if !self.viaConstructor {
		self.EnableLayouts = true
	}

	if len(self.AutolayoutPatterns) == 0 {
		self.AutolayoutPatterns = DefaultAutolayoutPatterns
	}

	if len(self.AutocompressPatterns) == 0 {
		self.AutocompressPatterns = DefaultAutocompressPatterns
	}

	if len(self.TemplatePatterns) == 0 {
		self.TemplatePatterns = DefaultTemplatePatterns
	}

	if len(self.RendererMappings) == 0 {
		self.RendererMappings = DefaultRendererMappings
	}

	if len(self.TryExtensions) == 0 {
		self.TryExtensions = DefaultTryExtensions
	}

	if len(self.FilterEnvVars) == 0 {
		self.FilterEnvVars = DefaultFilterEnvVars
	}

	if self.Address == `` {
		self.Address = DefaultAddress
	}

	if self.ErrorsPath == `` {
		self.ErrorsPath = DefaultErrorsPath
	}

	if self.IndexFile == `` {
		self.IndexFile = DefaultIndexFile
	}

	if self.LayoutPath == `` {
		self.LayoutPath = DefaultLayoutsPath
	}

	if self.RoutePrefix == `` {
		self.RoutePrefix = DefaultRoutePrefix
	}

	if self.VerifyFile == `` {
		self.VerifyFile = DefaultVerifyFile
	}

	if self.AutoindexTemplate == `` {
		self.AutoindexTemplate = DefaultAutoindexFilename
	}

	if self.Protocol == `` {
		self.Protocol = DefaultProtocol
	}

	if self.BindingTimeout == `` {
		self.BindingTimeout = DefaultBindingTimeout
	}

	if len(self.JWT) == 0 {
		self.JWT = make(map[string]*JWTConfig)
	}
}

func (self *Server) Initialize() error {
	self.populateDefaults()

	// if we haven't explicitly set a filesystem, create it
	if self.fs == nil {
		if strings.Contains(self.RootPath, `://`) {
			if mnt, err := NewMountFromSpec(`/:` + self.RootPath); err == nil {
				self.SetFileSystem(mnt)
			} else {
				return fmt.Errorf("root mount: %v", err)
			}
		} else {
			if v, err := fileutil.ExpandUser(self.RootPath); err == nil {
				self.RootPath = v
			}

			if v, err := filepath.Abs(self.RootPath); err == nil {
				self.RootPath = v
			} else {
				return fmt.Errorf("root path: %v", err)
			}

			self.SetFileSystem(http.Dir(self.RootPath))
		}
	}

	log.Debugf("rootfs: %T(%v)", self.fs, self.RootPath)

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

	if err := self.initJaegerTracing(); err != nil {
		return fmt.Errorf("jaeger: %v", err)
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

	if err := self.Bindings.init(self); err != nil {
		return fmt.Errorf("async bindings: %v", err)
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

func (self *Server) initJaegerTracing() error {
	// if enabled, initialize tracing (Jaeger/OpenTracing)
	if jc := self.JaegerConfig; jc != nil && jc.Enable {
		if cfg, err := jaegercfg.FromEnv(); err == nil {
			self.jaegerCfg = cfg
		} else {
			return fmt.Errorf("config: %v", err)
		}

		if self.jaegerCfg.ServiceName == `` {
			self.jaegerCfg.ServiceName = sliceutil.OrString(jc.ServiceName, `diecast`)
		}

		if self.jaegerCfg.Sampler == nil {
			self.jaegerCfg.Sampler = new(jaegercfg.SamplerConfig)
		}

		if self.jaegerCfg.Reporter == nil {
			self.jaegerCfg.Reporter = new(jaegercfg.ReporterConfig)
		}

		if r := self.jaegerCfg.Reporter; r != nil {
			if jc.QueueSize > 0 {
				r.QueueSize = jc.QueueSize
			}

			if jc.Agent != `` {
				r.LocalAgentHostPort = jc.Agent
			}

			if jc.Collector != `` {
				r.CollectorEndpoint = jc.Collector
			}

			if jc.Username != `` {
				r.User = jc.Username
			}

			if jc.Password != `` {
				r.Password = jc.Password
			}
		}

		if s := self.jaegerCfg.Sampler; s != nil {
			if s.Type == `` {
				s.Type = jaeger.SamplerTypeConst
				s.Param = 1
			} else if jc.SamplingType != `` {
				s.Type = jc.SamplingType
				s.Param = jc.SamplingParam
			}

			if jc.SamplingServerURL != `` {
				s.SamplingServerURL = jc.SamplingServerURL
			}

			if jc.SamplingMaxOperations > 0 {
				s.MaxOperations = jc.SamplingMaxOperations
			}

			if jc.SamplingRefreshInterval != `` {
				s.SamplingRefreshInterval = typeutil.Duration(jc.SamplingRefreshInterval)
			}
		}

		if jc.FlushInterval != `` {
			if bfi := typeutil.Duration(jc.FlushInterval); bfi >= (1 * time.Millisecond) {
				self.jaegerCfg.Reporter.BufferFlushInterval = bfi
			} else {
				return fmt.Errorf("invalid flush interval (minimum: 1ms)")
			}
		}

		if len(jc.Tags) > 0 {
			for k, v := range jc.Tags {
				self.jaegerCfg.Tags = append(self.jaegerCfg.Tags, opentracing.Tag{
					Key:   k,
					Value: v,
				})
			}
		}

		self.jaegerCfg.Tags = append(self.jaegerCfg.Tags, opentracing.Tag{
			Key:   `diecast-version`,
			Value: ApplicationVersion,
		})

		if ott, otc, err := self.jaegerCfg.NewTracer(); err == nil {
			self.opentrace = ott
			self.otcloser = otc

			opentracing.SetGlobalTracer(self.opentrace)

			var logline string

			if v := self.jaegerCfg.Reporter.CollectorEndpoint; v != `` {
				logline = fmt.Sprintf("collector at %s", v)
			} else if v := self.jaegerCfg.Reporter.LocalAgentHostPort; v != `` {
				logline = fmt.Sprintf("agent at %s", v)
			}

			if logline != `` {
				log.Debugf("trace: Jaeger tracing enabled: service=%s send to %s", self.jaegerCfg.ServiceName, logline)

				if len(self.jaegerCfg.Tags) > 0 {
					log.Debugf("trace: global tags:")
					for _, tag := range self.jaegerCfg.Tags {
						log.Debugf("trace: + %s: %v", tag.Key, typeutil.V(tag.Value))
					}
				}
			}
		} else {
			return err
		}
	}

	return nil
}

// Perform an end-to-end render of a single path, writing the output to the given writer,
// then exit.
func (self *Server) RenderPath(w io.Writer, path string) error {
	path = `/` + strings.TrimPrefix(path, `/`)

	var rw = httptest.NewRecorder()
	var req = httptest.NewRequest(http.MethodGet, path, nil)
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

// Perform a single request to the server and return an http.Response.
func (self *Server) GetResponse(method string, path string, body io.Reader, params map[string]interface{}, headers map[string]interface{}) *http.Response {
	path = `/` + strings.TrimPrefix(path, `/`)

	var rw = httptest.NewRecorder()
	var req = httptest.NewRequest(method, path, body)

	for k, v := range params {
		httputil.SetQ(req.URL, k, v)
	}

	for k, v := range headers {
		req.Header.Set(k, typeutil.String(v))
	}

	self.ServeHTTP(rw, req)

	if !rw.Flushed {
		rw.Flush()
	}

	return rw.Result()
}

// Return a URL string that can be used to perform requests from the local machine.
func (self *Server) LocalURL() string {
	return self.bestInternalLoopbackUrl(nil)
}

// Start a long-running webserver.  If provided, the functions provided will be run in parallel
// after the server has started.  If any of them return a non-nil error, the server will stop and
// this method will return that error.
func (self *Server) Serve(workers ...ServeFunc) error {
	var serveable Serveable
	var useTLS bool
	var useUDP bool
	var useSocket string
	var servechan = make(chan error)

	// fire off some goroutines for the prestart and start commands (if configured)
	if err := self.prestart(); err != nil {
		return err
	}

	var srv = &http.Server{
		Handler: self,
	}

	// work out if we're starting a UNIX socket server
	if addr := self.Address; strings.HasPrefix(addr, `unix:`) {
		useSocket = strings.TrimPrefix(addr, `unix:`)

		if useSocket == `` {
			useSocket = `diecast.` + typeutil.String(os.Getpid()) + `.sock`
		}
	} else {
		srv.Addr = addr
	}

	// setup TLSConfig
	if ssl := self.TLS; ssl != nil && ssl.Enable {
		var tc = new(tls.Config)

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
		var h2s = new(http2.Server)

		if useTLS {
			http2.ConfigureServer(srv, h2s)
		} else {
			var hnd = srv.Handler
			srv.Handler = h2c.NewHandler(hnd, h2s)
		}

		serveable = srv

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

		var network = `unix`

		if useUDP {
			network = `unixpacket`
		}

		if _, err := os.Stat(useSocket); err == nil {
			if err := os.Remove(useSocket); err != nil {
				return err
			}
		}

		if listener, err := net.Listen(network, useSocket); err == nil {
			go func() {
				if useTLS {
					servechan <- serveable.ServeTLS(listener, self.TLS.CertFile, self.TLS.KeyFile)
				} else {
					servechan <- serveable.Serve(listener)
				}

				os.Remove(useSocket)
			}()
		} else {
			return fmt.Errorf("bad socket: %v", err)
		}
	} else {
		go func() {
			if useTLS {
				servechan <- serveable.ListenAndServeTLS(self.TLS.CertFile, self.TLS.KeyFile)
			} else {
				servechan <- serveable.ListenAndServe()
			}
		}()
	}

	if len(workers) > 0 {
		for _, worker := range workers {
			go func(w ServeFunc) {
				servechan <- w(self)
			}(worker)
		}
	}

	return <-servechan
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

	// perform rate limiting check
	if rl := self.RateLimit; rl != nil && rl.Enable && rl.Limit != `` {
		if self.rateLimiter == nil {
			var lim = ratelimit.CreateLimit(rl.Limit)
			self.rateLimiter = &lim
		}

		if err := self.rateLimiter.Hit(rl.KeyFor(req)); err != nil {
			var didPenalty bool

			// impose sleep penalty if specified
			if penalty := rl.Penalty; penalty != `` {
				if pd := typeutil.Duration(penalty); pd > 0 {
					time.Sleep(pd)
					didPenalty = true
				}
			}

			if !didPenalty {
				self.respondError(w, req, err, http.StatusTooManyRequests)
				return
			}
		}
	}

	// setup a ResponseWriter interceptor that catches status code and bytes written
	// but passes through the Body without buffering it (like httptest.ResponseRecorder does)
	var interceptor = intercept(w)
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
	var baseName = filepath.Base(requestPath)

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

// return whether the request path matches any of the configured AutocompressPatterns.
func (self *Server) shouldAutocompress(requestPath string) bool {
	var baseName = filepath.Base(requestPath)

	for _, pattern := range self.AutocompressPatterns {
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
	var baseName = filepath.Base(requestPath)

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
	templateData []byte,
	header *TemplateHeader,
	urlParams []KV,
	mimeType string,
) error {
	var fragments = make(FragmentSet, 0)
	var forceSkipLayout = false
	var layouts = make([]string, 0)

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

	var earlyData = self.requestToEvalData(req, header)

	// get a reference to a set of standard functions that won't have a scope yet
	var earlyFuncs = self.GetTemplateFunctions(earlyData, header)

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
					if layoutName, err := EvalInline(layoutName, nil, earlyFuncs); err == nil {
						if layoutFile, err := self.LoadLayout(layoutName); err == nil {
							if err := fragments.Parse(LayoutTemplateName, layoutFile); err != nil {
								return err
							}

							break
						} else if layoutName != `default` {
							// we don't care if the default layout is missing
							return err
						}
					} else {
						return fmt.Errorf("layout: %v", err)
					}
				}
			}
		}
	}

	// get the content template in place
	// NOTE: make SURE this happens after the layout is loaded. this ensures that the layout data
	//       and bindings are evaluated first, then are overridden/appended by the content data/bindings
	if err := fragments.Set(ContentTemplateName, header, templateData); err != nil {
		return err
	}

	// get the merged header from all layouts, includes, and the template we're rendering
	var finalHeader = fragments.Header(self)

	// add all includes
	if err := self.appendIncludes(&fragments, &finalHeader); err != nil {
		return err
	}

	// put any url route params in there too
	finalHeader.UrlParams = urlParams

	// render locale from template
	if locale, err := EvalInline(finalHeader.Locale, earlyData, earlyFuncs); err == nil {
		finalHeader.Locale = locale
	} else {
		return fmt.Errorf("locale: %v", err)
	}

	if funcs, data, err := self.GetTemplateData(req, &finalHeader); err == nil {
		var start = time.Now()
		var fallingThrough bool

	SwitchCaseLoop:
		// switches allow the template processing to be hijacked/redirected mid-evaluation
		// based on data already evaluated
		for i, swcase := range finalHeader.Switch {
			if swcase == nil {
				continue SwitchCaseLoop
			}

			if !swcase.IsFallback() {
				if fallingThrough {
					continue SwitchCaseLoop
				}

				// if a condition is specified, it must evaluate to a truthy value to proceed
				var cond string

				if c, err := EvalInline(swcase.Condition, data, funcs); err == nil {
					cond = c
				} else {
					return fmt.Errorf("switch: %v", err)
				}

				var checkType, checkTypeArg = stringutil.SplitPair(swcase.CheckType, `:`)

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
					if !typeutil.Bool(cond) {
						continue SwitchCaseLoop
					}
				default:
					return fmt.Errorf("unknown switch checktype %q", swcase.CheckType)
				}

				if swcase.Break {
					break SwitchCaseLoop

				} else if swcase.Fallthrough {
					fallingThrough = true
					continue SwitchCaseLoop

				} else if redir := swcase.Redirect; redir != nil {
					finalHeader.Redirect = redir
					break SwitchCaseLoop

				} else if swTemplate, err := self.fs.Open(swcase.UsePath); err == nil {
					if swHeader, swData, err := SplitTemplateHeaderContent(swTemplate); err == nil {
						if fh, err := finalHeader.Merge(swHeader); err == nil {
							log.Debugf("[%s] Switch case %d matched, switching to template %v", reqid(req), i, swcase.UsePath)
							// httputil.RequestSetValue(req, SwitchCaseKey, usePath)

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

		if redirect := finalHeader.Redirect; redirect != nil {
			if u, err := EvalInline(redirect.URL, data, funcs); err == nil {
				if strings.TrimSpace(u) != `` {
					w.Header().Set(`Location`, strings.TrimSpace(u))

					if redirect.Code > 0 {
						w.WriteHeader(redirect.Code)
					} else {
						w.WriteHeader(http.StatusMovedPermanently)
					}

					return nil
				}
			} else {
				return err
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
		if renderer, err := EvalInline(finalHeader.Renderer, data, funcs); err == nil {
			finalHeader.Renderer = renderer
		} else {
			return fmt.Errorf("renderer: %v", err)
		}

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
					var intercept = httptest.NewRecorder()

					err = baseRenderer.Render(intercept, req, renderOpts)
					var res = intercept.Result()
					renderOpts.MimeType = res.Header.Get(`Content-Type`)
					renderOpts.Input = res.Body
				} else {
					renderOpts.Input = ioutil.NopCloser(bytes.NewBuffer(templateData))
				}

				if err == nil {
					// run the final template render and return
					log.Debugf("[%s] renderer: %T", reqid(req), postTemplateRenderer)

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
	var funcs = make(FuncMap)

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
		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			return reqinfo.URL.Query
		}

		return make(map[string]interface{})
	}

	// fn cookie: return a cookie value
	funcs[`cookie`] = func(key interface{}, fallbacks ...interface{}) interface{} {
		if len(fallbacks) == 0 {
			fallbacks = []interface{}{``}
		}

		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			if cookie := reqinfo.Cookie(typeutil.String(key)); cookie != nil {
				if cookie.Value != nil {
					return cookie.Value
				}
			}
		}

		return fallbacks[0]
	}

	// fn qs: Return the value of query string parameter *key* in the current URL, or return *fallback*.
	funcs[`qs`] = func(key interface{}, fallbacks ...interface{}) interface{} {
		if len(fallbacks) == 0 {
			fallbacks = []interface{}{``}
		}

		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			if v, ok := reqinfo.URL.Query[typeutil.String(key)]; ok {
				if typeutil.String(v) != `` {
					return v
				}
			}
		}

		return fallbacks[0]
	}

	// fn headers: Return the value of the *header* HTTP request header from the request used to
	//             generate the current view.
	funcs[`headers`] = func(key string, fallbacks ...interface{}) string {
		if len(fallbacks) == 0 {
			fallbacks = []interface{}{``}
		}

		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			if v, ok := reqinfo.Headers[key]; ok {
				if vS := typeutil.String(v); vS != `` {
					return vS
				}
			}
		}

		return typeutil.String(fallbacks[0])
	}

	// fn param: Return the value of the named or indexed URL parameter, or nil of none are present.
	funcs[`param`] = func(nameOrIndex interface{}, fallbacks ...interface{}) interface{} {
		var params []KV

		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			params = reqinfo.URL.Params
		}

		for i, kv := range params {
			var wantKey = typeutil.String(nameOrIndex)
			var wantIndex = int(typeutil.Int(wantKey))

			if sval := typeutil.String(kv.V); sval != `` {
				if typeutil.IsInteger(wantKey) {
					// zero index: return all param values as an array
					if wantIndex == 0 {
						return kvValues(params)
					} else if i == (wantIndex - 1) {
						return typeutil.Auto(kv.V)
					}
				} else if kv.K == wantKey {
					return typeutil.Auto(kv.V)
				}
			}
		}

		if len(fallbacks) > 0 {
			return fallbacks[0]
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
		var path = makeVarKey(name)

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
		var key = makeVarKey(name)

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
		var key = makeVarKey(name)

		if existing := maputil.DeepGet(data, key); existing != nil {
			var values = sliceutil.Sliceify(existing)

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
	funcs[`increment`] = func(name string, incr ...interface{}) interface{} {
		var key = makeVarKey(name)
		var count float64
		var incrV float64 = typeutil.OrFloat(0, incr...)

		if existing := maputil.DeepGet(data, key); existing != nil {
			count = typeutil.V(existing).Float()
		}

		if incrV > 0 {
			count += incrV
		} else {
			count += 1
		}

		maputil.DeepSet(data, key, count)

		return ``
	}

	// fn incrementByValue: Add a number to a counter tracking the number of occurrences of a specific value.
	funcs[`incrementByValue`] = func(name string, value interface{}, incr ...interface{}) interface{} {
		var key = makeVarKey(name, fmt.Sprintf("%v", value))
		var count float64
		var incrV float64 = typeutil.OrFloat(0, incr...)

		if existing := maputil.DeepGet(data, key); existing != nil {
			count = typeutil.V(existing).Float()
		}

		if incrV > 0 {
			count += incrV
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
			var d = data

			if len(overrides) > 0 && overrides[0] != nil {
				d = overrides[0]
			}

			return EvalInline(string(tpl), d, funcs)
		} else {
			return ``, err
		}
	}

	funcs[`locale`] = func(fallback ...string) (string, error) {
		if header.Locale != `` {
			if tag, err := language.Parse(header.Locale); err == nil {
				if l := tag.String(); l != `` {
					return l, nil
				}
			} else {
				return ``, err
			}
		}

		if len(fallback) > 0 {
			return fallback[0], nil
		} else {
			return ``, nil
		}
	}

	funcs[`localeBase`] = func(fallback ...string) (string, error) {
		if header.Locale != `` {
			if tag, err := language.Parse(header.Locale); err == nil {
				if l := i18nTagBase(tag); l != `` {
					return l, nil
				}
			} else {
				return ``, err
			}
		}

		if len(fallback) > 0 {
			return fallback[0], nil
		} else {
			return ``, nil
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
		var kparts = strings.Split(key, `.`)

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
		var acceptLanguage string

		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			acceptLanguage = reqinfo.Header(`accept_language`).String()
		}
		if acceptLanguage != `` {
			if tags, _, err := language.ParseAcceptLanguage(acceptLanguage); err == nil {
				for _, tag := range tags {
					locales = append(locales, tag.String())
					locales = append(locales, i18nTagBase(tag))
				}
			} else {
				log.Warningf("i18n: invalid Accept-Language value %q", acceptLanguage)
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

	// fn jwt: generate a new JSON Web Token using the named configuration
	funcs[`jwt`] = func(jwtConfigName string) (string, error) {
		if len(self.JWT) > 0 {
			if cfg, ok := self.JWT[jwtConfigName]; ok && cfg != nil {
				return cfg.Issue(data, funcs)
			}
		}

		return ``, fmt.Errorf("JWT configuration %q not found", jwtConfigName)
	}

	// fn jwtSign: generates a signature for the given input.  If the input is a map or array, it will be
	// JSON-encoded before signing.
	funcs[`jwtSign`] = func(jwtConfigName string, input interface{}) (string, error) {
		if len(self.JWT) > 0 {
			if cfg, ok := self.JWT[jwtConfigName]; ok && cfg != nil {
				if typeutil.IsScalar(input) {
					return cfg.SignedString(typeutil.String(input))
				} else {
					return cfg.SignedString(typeutil.JSON(input))
				}
			}
		}

		return ``, fmt.Errorf("JWT configuration %q not found", jwtConfigName)
	}

	funcs[`jwtSecret`] = func(jwtConfigName string) (string, error) {
		if len(self.JWT) > 0 {
			if cfg, ok := self.JWT[jwtConfigName]; ok && cfg != nil {
				return EvalInline(cfg.Secret, data, funcs)
			}
		}

		return ``, fmt.Errorf("JWT configuration %q not found", jwtConfigName)
	}

	return funcs
}

func makeVarKey(key string, post ...string) []string {
	var output = []string{`vars`}

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
	var data = self.requestToEvalData(req, header)
	var funcs = self.GetTemplateFunctions(data, header)

	data[`vars`] = make(map[string]interface{})

	var publicMountDetails = make([]map[string]interface{}, 0)

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

	self.evalPageData(false, req, header, funcs, data)

	return funcs, data
}

func (self *Server) evalPageData(final bool, req *http.Request, header *TemplateHeader, funcs FuncMap, data map[string]interface{}) map[string]interface{} {
	// Evaluate "page" data: this data is templatized, but does not have access
	//                       to the output of bindings
	// ---------------------------------------------------------------------------------------------
	var pageData = make(map[string]interface{})

	if header != nil {
		var applyPageFn = func(value interface{}, path []string, isLeaf bool) error {

			if isLeaf {
				switch value.(type) {
				case string:
					if v, err := EvalInline(value.(string), data, funcs); err == nil {
						value = v
					} else {
						return err
					}

					value = stringutil.Autotype(value)

					// not final pass + no value = leave the expression intact so that a later pass might succeed
					if !final && value == nil {
						return nil
					}
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
	}

	data[`page`] = pageData
	data[`p`] = pageData

	return pageData
}

func (self *Server) updateBindings(data map[string]interface{}, with map[string]interface{}) {
	data[`bindings`] = with
	data[`b`] = with
}

func (self *Server) GetTemplateData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]interface{}, error) {
	var funcs, data = self.getPreBindingData(req, header)

	// Evaluate "bindings": Bindings have access to $.page, and each subsequent binding has access
	//                      to all binding output that preceded it.  This allows bindings to be
	//                      pipelined, using the output of one request as the input of the next.
	// ---------------------------------------------------------------------------------------------
	var bindings = maputil.M(&self.sharedBindingData).MapNative()
	var bindingsToEval = make([]Binding, 0)

	// only use top-level bindings that
	bindingsToEval = append(bindingsToEval, self.Bindings.perRequestBindings()...)

	if header != nil {
		bindingsToEval = append(bindingsToEval, header.Bindings...)
	}

	for i, binding := range bindingsToEval {
		if strings.TrimSpace(binding.Name) == `` {
			binding.Name = fmt.Sprintf("binding%d", i)
		}

		binding.server = self

		var start = time.Now()
		describeTimer(fmt.Sprintf("binding-%s", binding.Name), fmt.Sprintf("Diecast Bindings: %s", binding.Name))

		if header != nil {
			if v, err := maputil.Merge(header.DefaultHeaders, binding.Headers); err == nil {
				binding.Headers = maputil.Stringify(v)
			} else {
				return nil, nil, fmt.Errorf("merge headers: %v", err)
			}
		}

		// pagination data
		if pgConfig := binding.Paginate; pgConfig != nil {
			var results = make([]map[string]interface{}, 0)
			var proceed = true
			var total int64
			var count int64
			var soFar int64
			var page = 1

			var lastPage = maputil.M(&ResultsPage{
				Page:    page,
				Counter: soFar,
			}).MapNative(`json`)

			for proceed {
				var suffix = fmt.Sprintf("binding(%s):page(%d)", binding.Name, page+1)

				bindings[binding.Name] = binding.Fallback
				data[`page`] = lastPage

				if len(binding.Params) == 0 {
					binding.Params = make(map[string]interface{})
				}

				// eval the URL
				if r, err := EvalInline(binding.Resource, data, funcs, suffix); err == nil {
					binding.Resource = r
				} else {
					return nil, nil, fmt.Errorf("resource: %v", err)
				}

				// eval / set querystring params
				for qsk, qsv := range pgConfig.QueryStrings {
					if t, err := EvalInline(qsv, data, funcs, suffix); err == nil {
						binding.Params[qsk] = typeutil.Auto(t)
					} else {
						return nil, nil, fmt.Errorf("param: %v", err)
					}
				}

				// eval / set request headers
				for hk, hv := range pgConfig.Headers {
					if t, err := EvalInline(hv, data, funcs, suffix); err == nil {
						binding.Headers[hk] = t
					} else {
						return nil, nil, fmt.Errorf("headers: %v", err)
					}
				}

				v, err := binding.tracedEvaluate(req, header, data, funcs)

				if err == nil {
					var asMap = maputil.M(v)

					if v, err := EvalInline(pgConfig.Total, asMap.MapNative(), funcs, suffix); err == nil {
						total = typeutil.Int(v)
					} else {
						return nil, nil, fmt.Errorf("paginate: %v", err)
					}

					if v, err := EvalInline(pgConfig.Count, asMap.MapNative(), funcs, suffix); err == nil {
						count = typeutil.Int(v)
					} else {
						return nil, nil, fmt.Errorf("paginate: %v", err)
					}

					soFar += count

					log.Debugf("[%v] paginated binding %q: total=%v count=%v soFar=%v", reqid(req), binding.Name, total, count, soFar)

					if v, err := EvalInline(pgConfig.Done, asMap.MapNative(), funcs, suffix); err == nil {
						proceed = !typeutil.Bool(v)
					} else {
						return nil, nil, fmt.Errorf("paginate: %v", err)
					}

					if pgConfig.Maximum > 0 && soFar >= pgConfig.Maximum {
						proceed = false
					}

					if !proceed {
						log.Debugf("[%v] paginated binding %q: proceed is false, this is the last loop", reqid(req), binding.Name)
					}

					var thisPage = maputil.M(&ResultsPage{
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
					self.updateBindings(data, bindings)
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

				self.updateBindings(data, bindings)
				page++
			}

			bindings[binding.Name] = results
			self.updateBindings(data, bindings)

		} else if binding.Repeat == `` {
			bindings[binding.Name] = binding.Fallback
			self.updateBindings(data, bindings)

			v, err := binding.tracedEvaluate(req, header, data, funcs)

			if err == nil && v != nil {
				bindings[binding.Name] = v
				self.updateBindings(data, bindings)
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
			var results = make([]interface{}, 0)

			var repeatExpr = fmt.Sprintf("{{ range $index, $item := (%v) }}\n", binding.Repeat)
			repeatExpr += fmt.Sprintf("%v\n", binding.Resource)
			repeatExpr += "{{ end }}"
			var repeatExprOut string

			if v, err := EvalInline(repeatExpr, data, funcs); err == nil {
				repeatExprOut = rxEmptyLine.ReplaceAllString(strings.TrimSpace(v), ``)
			} else {
				return nil, nil, fmt.Errorf("repeater: %v", err)
			}

			log.Debugf("Repeater: \n%v\nOutput:\n%v", repeatExpr, repeatExprOut)
			var repeatIters = strings.Split(repeatExprOut, "\n")

			for i, resource := range repeatIters {
				binding.Resource = strings.TrimSpace(resource)
				binding.Repeat = ``

				bindings[binding.Name] = binding.Fallback

				var v, err = binding.tracedEvaluate(req, header, data, funcs)

				if err == nil {
					results = append(results, v)
					bindings[binding.Name] = results
					self.updateBindings(data, bindings)
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

				self.updateBindings(data, bindings)
			}

		}

		reqtime(req, fmt.Sprintf("binding-%s", binding.Name), time.Since(start))

		// re-evaluate page based on new binding results
		self.evalPageData(false, req, header, funcs, data)
	}

	self.updateBindings(data, bindings)

	// Evaluate "flags" data: this data is templatized, and has access to $.page and $.bindings
	// ---------------------------------------------------------------------------------------------
	if header != nil {
		var flags = make(map[string]bool)

		for name, def := range header.FlagDefs {
			switch def.(type) {
			case bool:
				flags[name] = def.(bool)
			default:
				if flag, err := EvalInline(fmt.Sprintf("%v", def), data, funcs); err == nil {
					flags[name] = typeutil.V(flag).Bool()
				} else {
					return nil, nil, fmt.Errorf("flags: %v", err)
				}
			}
		}

		data[`flags`] = flags
	}

	// the final pass on page; any empty values resulting from this are final
	self.evalPageData(true, req, header, funcs, data)

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
				return file, ``, DirectoryErr
			}
		} else {
			return file, ``, fmt.Errorf("failed to stat file %v: %v", requestPath, err)
		}
	} else {
		return nil, ``, err
	}
}

// Try to load the given path from each of the mounts, and return the matching mount and its response
// if found.
func (self *Server) tryMounts(requestPath string, req *http.Request) (Mount, *MountResponse, error) {
	var body *RequestBody

	if rb := reqbody(req); rb != nil {
		body = rb
	} else {
		return nil, nil, fmt.Errorf("no request body")
	}

	var lastErr error

	// find a mount that has this file
	for _, mount := range self.Mounts {
		// closing the RequestBody resets the reader to the beginning
		body.Close()

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

	body.Close()

	if lastErr == nil {
		lastErr = fmt.Errorf("%q not found", requestPath)
	}

	return nil, nil, lastErr
}

func (self *Server) respondError(w http.ResponseWriter, req *http.Request, resErr error, code int) {
	var tmpl = NewTemplate(`error`, HtmlEngine)

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

				if code >= 400 {
					w.WriteHeader(code)
				}

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
		// chop off shebang line
		if bytes.HasPrefix(data, []byte("#!")) {
			// first possible position for a \n after the shebang is nl=2
			if nl := bytes.Index(data, []byte("\n")); nl > 1 {
				if (nl + 1) < len(data) {
					data = data[nl+1:]
				}
			}
		}

		if bytes.HasPrefix(data, HeaderSeparator) {
			var parts = bytes.SplitN(data, HeaderSeparator, 3)

			if len(parts) == 3 {
				var header = TemplateHeader{
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

func reqbody(req *http.Request) *RequestBody {
	if body, ok := httputil.RequestGetValue(req, RequestBodyKey).Value.(*RequestBody); ok {
		return body
	}

	return nil
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
	var route = req.URL.Path

	for _, action := range self.Actions {
		var actionPath = filepath.Join(self.rp(), action.Path)

		if actionPath == route {
			var methods = sliceutil.Stringify(action.Method)

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
	var rv = map[string]interface{}{}

	var request = RequestInfo{
		Headers: make(map[string]interface{}),
		Cookies: make(map[string]Cookie),
		URL: RequestUrlInfo{
			Query: make(map[string]interface{}),
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

	// cookie values
	// ------------------------------------------------------------------------
	for _, cookie := range req.Cookies() {
		if _, ok := request.Cookies[cookie.Name]; !ok {
			request.Cookies[cookie.Name] = Cookie{
				Name:     cookie.Name,
				Value:    typeutil.Auto(cookie.Value),
				Path:     cookie.Path,
				Domain:   cookie.Domain,
				MaxAge:   &cookie.MaxAge,
				Secure:   &cookie.Secure,
				HttpOnly: &cookie.HttpOnly,
				SameSite: MakeCookieSameSite(cookie.SameSite),
			}
		}
	}

	for k, v := range req.URL.Query() {
		if vv := strings.Join(v, qj); !typeutil.IsZero(vv) {
			request.URL.Query[k] = stringutil.Autotype(vv)
		}
	}

	// request headers
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
	request.Body = reqbody(req)
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

		var sslclients = make([]RequestTlsCertInfo, 0)

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

	if m, err := request.asMap(); err == nil {
		rv[`request`] = m
		rv[`_request`] = &request

		// top-level shortcuts to otherwise more syntax-intensive common cases
		rv[`r`] = m                  // request data
		rv[`qs`] = request.URL.Query // query strings
		rv[`h`] = request.Headers    // request headers
		rv[`c`] = request.Cookies    // cookies
	} else {
		panic(err.Error())
	}

	// environment variables
	var env = make(map[string]interface{})

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

				var env = make(map[string]interface{})

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
					var waitchan = make(chan error)

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
		var format = logFormats[self.Log.Format]

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

		var interceptor = reqres(req)
		rh, rp := stringutil.SplitPair(req.RemoteAddr, `:`)
		var code = typeutil.String(interceptor.code)

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

		var logContext = maputil.M(map[string]interface{}{
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

func (self *Server) bindingTimeout() time.Duration {
	if t := typeutil.Duration(self.BindingTimeout); t > 0 {
		return t
	} else {
		return DefaultBindingTimeout
	}
}

func (self *Server) bestInternalLoopbackUrl(req *http.Request) string {
	if self.BindingPrefix != `` {
		return self.BindingPrefix
	}

	var proto string

	if self.TLS != nil && self.TLS.Enable {
		proto = `https`
	} else {
		proto = `http`
	}

	if strings.HasPrefix(self.Address, `unix:`) {
		var path = self.Address

		path = strings.TrimPrefix(path, `unix:`)
		path = strings.ReplaceAll(path, `/`, weirdPathsInHostnamesPlaceholder)

		return proto + `+unix://` + path

	} else if h, p, err := net.SplitHostPort(self.Address); err == nil {
		switch h {
		case `0.0.0.0`, `::/0`, `[::/0]`:
			return proto + `://localhost:` + p
		}
	}

	if req != nil && req.Host != `` {
		return proto + `://` + req.Host
	} else {
		return proto + `://` + self.Address
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

	var url = fmt.Sprintf("%s %v %v", req.Method, req.URL, req.Proto)

	request = append(request, url)
	request = append(request, fmt.Sprintf("host: %s", req.Host))
	var headerNames = maputil.StringKeys(req.Header)
	sort.Strings(headerNames)

	for _, name := range headerNames {
		var headers = req.Header[name]
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
