package diecast

//go:generate make favicon.go

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"errors"
	"fmt"
	"html/template"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
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

//go:embed ui
var embedded embed.FS

var ITotallyUnderstandRunningArbitraryCommandsAsRootIsRealRealBad = false
var ErrIsDirectory = errors.New(`is a directory`)
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
	return (err == ErrIsDirectory)
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

func (server RedirectTo) Error() string {
	return string(server)
}

type StartCommand struct {
	Command          string         `yaml:"command"          json:"command"`          // The shell command line to execute on start
	Directory        string         `yaml:"directory"        json:"directory"`        // The working directory the command should be run from
	Environment      map[string]any `yaml:"env"              json:"env"`              // A map of environment variables to expose to the command
	WaitBefore       string         `yaml:"delay"            json:"delay"`            // How long to delay before running the command
	Wait             string         `yaml:"timeout"          json:"timeout"`          // How long to wait before killing the command
	ExitOnCompletion bool           `yaml:"exitOnCompletion" json:"exitOnCompletion"` // Whether Diecast should exit upon command completion
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
	Disable     bool   `yaml:"disable"              json:"disable"`     // if true, no log output will be written
}

type RateLimitConfig struct {
	Enable    bool   `yaml:"enable"     json:"enable"`
	Limit     string `yaml:"limit"      json:"limit"`      // Specify a rate limit string (e.g.: "1r/s", "200r/m")
	PerClient bool   `yaml:"per_client" json:"per_client"` // Specify that the limit should be applied per-client instead of globally.
	Penalty   string `yaml:"penalty"    json:"penalty"`    // An amount of time to sleep instead of returning an HTTP 429 error on rate limited requests
}

func (server *RateLimitConfig) KeyFor(req *http.Request) string {
	if server.PerClient {
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

func (server *TraceMapping) TraceName(candidate string) (string, bool) {
	if server.Match != `` {
		if server.rx == nil {
			if rx, err := regexp.Compile(server.Match); err == nil {
				server.rx = rx
			}
		}
	}

	if server.rx != nil {
		if server.rx.MatchString(candidate) {
			return server.rx.ReplaceAllString(candidate, server.Replace), true
		}
	}

	return ``, false
}

type JWTConfig struct {
	Algorithm string         `yaml:"alg"     json:"alg"`     // The JWT signing algorithm to use (default: HS256)
	Secret    string         `yaml:"secret"  json:"secret"`  // The JWT secret used to sign payloads
	Claims    map[string]any `yaml:"claims"  json:"claims"`  // The claims being made (i.e.: the payload that will be converted to JSON)
	Expires   any            `yaml:"expires" json:"expires"` // A duration string representing how long issued tokens will be valid for (default: 60s)
	Issuer    string         `yaml:"issuer"  json:"issuer"`  // The JWT issuer
	Subject   string         `yaml:"subject" json:"subject"`
}

func (server *JWTConfig) SignedString(input string) (string, error) {
	var alg = typeutil.OrString(server.Algorithm, `HS256`)

	if signer := jwt.GetSigningMethod(alg); signer != nil {
		return signer.Sign(input, []byte(server.Secret))
	} else {
		return ``, fmt.Errorf("invalid signing algorithm %q", alg)
	}
}

func (server *JWTConfig) Issue(tpldata map[string]any, funcs FuncMap) (string, error) {
	var now = time.Now()
	var alg = typeutil.OrString(server.Algorithm, `HS256`)
	var expiry = typeutil.OrDuration(server.Expires, `60s`)

	if signer := jwt.GetSigningMethod(alg); signer != nil {
		var claims jwt.Claims

		if len(server.Claims) == 0 {
			claims = jwt.StandardClaims{
				Id:        stringutil.UUID().String(),
				IssuedAt:  now.Unix(),
				ExpiresAt: now.Add(expiry).Unix(),
				Issuer:    ShouldEvalInline(server.Issuer, tpldata, funcs).String(),
				Subject:   ShouldEvalInline(server.Subject, tpldata, funcs).String(),
				NotBefore: now.Unix(),
			}
		} else {
			var c = jwt.MapClaims(server.Claims)

			if server.Issuer != `` {
				c[`iss`] = server.Issuer
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
		return jwt.NewWithClaims(signer, claims).SignedString([]byte(server.Secret))
	} else {
		return ``, fmt.Errorf("invalid signing algorithm %q", alg)
	}
}

type JaegerConfig struct {
	Enable                  bool            `yaml:"enable"                  json:"enable"`                  // Explicitly enable or disable Jaeger tracing
	ServiceName             string          `yaml:"service"                 json:"service"`                 // Set the service name that traces will fall under.
	Agent                   string          `yaml:"agent"                   json:"agent"`                   // Specify the host:port of a local UDP agent to send traces to.
	Collector               string          `yaml:"collector"               json:"collector"`               // Specify the collector address to sent traces to.  Overrides "agent" if set.
	Username                string          `yaml:"username"                json:"username"`                // Provides a username to authenticate with the collector.
	Password                string          `yaml:"password"                json:"password"`                // Provides a password to authenticate with the collector.
	QueueSize               int             `yaml:"queueSize"               json:"queueSize"`               // Specify the size of the queue for outgoing reports.
	FlushInterval           string          `yaml:"flushInterval"           json:"flushInterval"`           // Duration specifying how frequently queued reports should be flushed.
	Tags                    map[string]any  `yaml:"tags"                    json:"tags"`                    // A set of key-value pairs that are included in every trace.
	SamplingType            string          `yaml:"sampling"                json:"sampling"`                // Specifies the type of sampling to use: const, probabilistic, rateLimiting, or remote.
	SamplingParam           float64         `yaml:"samplingParam"           json:"samplingParam"`           // A type-specific parameter used to configure that type of sampling; const: 0 or 1, probabilistic: 0.0-1.0, rateLimiting: max number of spans per seconds, remote: same as probabilistic.
	SamplingServerURL       string          `yaml:"samplingUrl"             json:"samplingUrl"`             // The sampling server URL for the "remote" sampling type.
	SamplingRefreshInterval string          `yaml:"samplingRefreshInterval" json:"samplingRefreshInterval"` // How frequently to poll the remote sampling server.
	SamplingMaxOperations   int             `yaml:"samplingMaxOps"          json:"samplingMaxOps"`          // A maximum number of operations for certain sampling modes.
	OperationsMappings      []*TraceMapping `yaml:"operations"              json:"operations"`              // Maps regular expressions used to match specific routes to the operation name that will be emitted in traces. Without a matching expression, traces will be named by the calling HTTP method and Request URI.  The string being tested by these regular expressions is the one that would be emitted otherwise; so "GET /path/to/file"
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
	DefaultPageObject    map[string]any            `yaml:"-"                       json:"-"`                       //
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
	GlobalHeaders        map[string]any            `yaml:"globalHeaders,omitempty" json:"globalHeaders,omitempty"` // A set of HTTP headers that should be added to EVERY response Diecast returns, regardless of whether it originates from a template, mount, or other configuration.
	IndexFile            string                    `yaml:"indexFile"               json:"indexFile"`               // The name of the template file to use when a directory is requested.
	LayoutPath           string                    `yaml:"layouts"                 json:"layouts"`                 // The path to the layouts template directory
	Locale               string                    `yaml:"locale"                  json:"locale"`                  // Specify the default locale for pages being served.
	MountConfigs         []MountConfig             `yaml:"mounts"                  json:"mounts"`                  // A list of mount configurations read from the diecast.yml config file.
	Mounts               []Mount                   `yaml:"-"                       json:"-"`                       // The set of all registered mounts.
	OnAddHandler         AddHandlerFunc            `yaml:"-"                       json:"-"`                       // A function that can be used to intercept handlers being added to the server.
	OverridePageObject   map[string]any            `yaml:"-"                       json:"-"`                       //
	PrestartCommands     []*StartCommand           `yaml:"prestart"                json:"prestart"`                // A command that will be executed before the server is started.
	Protocols            map[string]ProtocolConfig `yaml:"protocols"               json:"protocols"`               // Setup global configuration details for Binding Protocols
	RendererMappings     map[string]string         `yaml:"rendererMapping"         json:"rendererMapping"`         // Map file extensions to preferred renderers for a given file type.
	RootPath             string                    `yaml:"root"                    json:"root"`                    // The filesystem location where templates and files are served from
	RoutePrefix          string                    `yaml:"routePrefix"             json:"routePrefix"`             // If specified, all requests must be prefixed with this string.
	StartCommands        []*StartCommand           `yaml:"start"                   json:"start"`                   // A command that will be executed after the server is confirmed running.
	TLS                  *TlsConfig                `yaml:"tls"                     json:"tls"`                     // where SSL/TLS configuration is stored
	TemplatePatterns     []string                  `yaml:"patterns"                json:"patterns"`                // A set of glob patterns specifying which files will be rendered as templates.
	Translations         map[string]any            `yaml:"translations,omitempty"  json:"translations,omitempty"`  // Stores translations for use with the i18n and l10n functions.  Keys values represent the
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
	BindingTimeout       any                       `yaml:"bindingTimeout"          json:"bindingTimeout"`          // Sets the default timeout for bindings that don't explicitly set one.
	JaegerConfig         *JaegerConfig             `yaml:"jaeger"                  json:"jaeger"`                  // Configures distributed tracing using Jaeger.
	AutocompressPatterns []string                  `yaml:"autocompress"            json:"autocompress"`            // A set of glob patterns indicating directories whose contents will be delivered as ZIP files
	RequestBodyPreload   int64                     `yaml:"requestPreload"          json:"requestPreload"`          // Maximum number of bytes to read from a request body for the purpose of automatically parsing it.  Requests larger than this will not be available to templates.
	JWT                  map[string]*JWTConfig     `yaml:"jwt"                     json:"jwt"`                     // Contains configurations for generating JSON Web Tokens in templates.
	altRootCaPool        *x509.CertPool
	faviconImageIco      []byte
	fs                   http.FileSystem
	hasUserRoutes        bool
	initialized          bool
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

func NewServer(root any, patterns ...string) *Server {
	describeTimer(`tpl`, `Diecast Template Rendering`)

	var server = &Server{
		RootPath:           `.`,
		TemplatePatterns:   patterns,
		Authenticators:     make(AuthenticatorConfigs, 0),
		Bindings:           make(SharedBindingSet, 0),
		DefaultPageObject:  make(map[string]any),
		Mounts:             make([]Mount, 0),
		OverridePageObject: make(map[string]any),
		GlobalHeaders:      make(map[string]any),
		EnableLayouts:      true,
		RequestBodyPreload: DefaultRequestBodyPreload,
		mux:                http.NewServeMux(),
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

func (server *Server) ShouldReturnSource(req *http.Request) bool {
	if server.EnableDebugging {
		if httputil.QBool(req, DebuggingQuerystringParam) {
			return true
		}
	}

	return false
}

func (server *Server) LoadConfig(filename string) error {
	if pathutil.FileExists(filename) {
		if file, err := os.Open(filename); err == nil {
			defer file.Close()
			return server.LoadConfigFromReader(file, filename)
		} else {
			return err
		}
	}

	return nil
}

func (server *Server) LoadConfigFromReader(file io.Reader, filename string) error {
	if data, err := io.ReadAll(file); err == nil && len(data) > 0 {
		data = []byte(stringutil.ExpandEnv(string(data)))

		if err := yaml.UnmarshalStrict(data, server); err == nil {
			// apply environment-specific overrides
			if server.Environment != `` && filename != `` {
				eDir, eFile := filepath.Split(filename)
				var base = strings.TrimSuffix(eFile, filepath.Ext(eFile))
				var ext = filepath.Ext(eFile)
				eFile = fmt.Sprintf("%s.%s%s", base, server.Environment, ext)
				var envPath = filepath.Join(eDir, eFile)

				if fileutil.IsNonemptyFile(envPath) {
					if err := server.LoadConfig(envPath); err != nil {
						return fmt.Errorf("failed to load %s: %v", eFile, err)
					}
				}
			}

			// process mount configs into mount instances
			for i, config := range server.MountConfigs {
				if mount, err := NewMountFromSpec(fmt.Sprintf("%s:%s", config.Mount, config.To)); err == nil {
					var mountOverwriteIndex = -1

					for i, existing := range server.Mounts {
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
						server.Mounts[mountOverwriteIndex] = mount
					} else {
						server.Mounts = append(server.Mounts, mount)
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
func (server *Server) SetMounts(mounts []Mount) {
	if len(server.Mounts) > 0 {
		server.Mounts = append(server.Mounts, mounts...)
	} else {
		server.Mounts = mounts
	}
}

func (server *Server) SetFileSystem(fs http.FileSystem) {
	server.fs = fs
}

// Read a file from the underlying root filesystem, satisfying the http.FileSystem interface.
func (server *Server) Open(name string) (http.File, error) {
	if server.fs == nil {
		return nil, fmt.Errorf("no filesystem")
	} else {
		return server.fs.Open(name)
	}
}

func (server *Server) IsInRootPath(path string) bool {
	if absR, err := filepath.Abs(server.RootPath); err == nil {
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

func (server *Server) populateDefaults() {
	if server.mux == nil {
		server.mux = http.NewServeMux()
	}

	server.handlersEnsureRouter()

	if server.Log.Format == `` {
		server.Log.Format = logFormats[`common`]
		server.Log.Destination = `-`
		server.Log.Colorize = true
	}

	if !server.viaConstructor {
		server.EnableLayouts = true
	}

	if len(server.AutolayoutPatterns) == 0 {
		server.AutolayoutPatterns = DefaultAutolayoutPatterns
	}

	if len(server.AutocompressPatterns) == 0 {
		server.AutocompressPatterns = DefaultAutocompressPatterns
	}

	if len(server.TemplatePatterns) == 0 {
		server.TemplatePatterns = DefaultTemplatePatterns
	}

	if len(server.RendererMappings) == 0 {
		server.RendererMappings = DefaultRendererMappings
	}

	if len(server.TryExtensions) == 0 {
		server.TryExtensions = DefaultTryExtensions
	}

	if len(server.FilterEnvVars) == 0 {
		server.FilterEnvVars = DefaultFilterEnvVars
	}

	if server.Address == `` {
		server.Address = DefaultAddress
	}

	if server.ErrorsPath == `` {
		server.ErrorsPath = DefaultErrorsPath
	}

	if server.IndexFile == `` {
		server.IndexFile = DefaultIndexFile
	}

	if server.LayoutPath == `` {
		server.LayoutPath = DefaultLayoutsPath
	}

	if server.RoutePrefix == `` {
		server.RoutePrefix = DefaultRoutePrefix
	}

	if server.VerifyFile == `` {
		server.VerifyFile = DefaultVerifyFile
	}

	if server.AutoindexTemplate == `` {
		server.AutoindexTemplate = DefaultAutoindexFilename
	}

	if server.Protocol == `` {
		server.Protocol = DefaultProtocol
	}

	if server.BindingTimeout == `` {
		server.BindingTimeout = DefaultBindingTimeout
	}

	if len(server.JWT) == 0 {
		server.JWT = make(map[string]*JWTConfig)
	}
}

func (server *Server) Initialize() error {
	server.populateDefaults()

	// if we haven't explicitly set a filesystem, create it
	if server.fs == nil {
		if strings.Contains(server.RootPath, `://`) {
			if mnt, err := NewMountFromSpec(`/:` + server.RootPath); err == nil {
				server.SetFileSystem(mnt)
			} else {
				return fmt.Errorf("root mount: %v", err)
			}
		} else {
			if v, err := fileutil.ExpandUser(server.RootPath); err == nil {
				server.RootPath = v
			}

			if v, err := filepath.Abs(server.RootPath); err == nil {
				server.RootPath = v
			} else {
				return fmt.Errorf("root path: %v", err)
			}

			server.SetFileSystem(http.Dir(server.RootPath))
		}
	}

	log.Debugf("rootfs: %T(%v)", server.fs, server.RootPath)

	// allocate ephemeral address if we're supposed to
	if addr, port, err := net.SplitHostPort(server.Address); err == nil {
		if port == `0` {
			if allocated, err := netutil.EphemeralPort(); err == nil {
				server.Address = fmt.Sprintf("%v:%d", addr, allocated)
			} else {
				return err
			}
		}
	}

	if err := server.initJaegerTracing(); err != nil {
		return fmt.Errorf("jaeger: %v", err)
	}

	// if configured, this path must exist (relative to RootPath or the root filesystem) or Diecast will refuse to start
	if server.VerifyFile != `` {
		if verify, err := server.fs.Open(server.VerifyFile); err == nil {
			verify.Close()
		} else {
			return fmt.Errorf("failed to open verification file %q: %v", server.VerifyFile, err)
		}
	}

	if err := server.setupServer(); err != nil {
		return err
	}

	if err := server.Bindings.init(server); err != nil {
		return fmt.Errorf("async bindings: %v", err)
	}

	server.initialized = true

	if server.DisableCommands {
		log.Noticef("Not executing PrestartCommand because DisableCommands is set")
		return nil
	} else if _, err := server.RunStartCommand(server.PrestartCommands, false); err != nil {
		return err
	} else {
		return nil
	}
}

func (server *Server) prestart() error {
	if !server.initialized {
		if err := server.Initialize(); err != nil {
			return err
		}
	}

	go func() {
		if server.DisableCommands {
			log.Noticef("Not executing StartCommand because DisableCommands is set")
			return
		}

		eoc, err := server.RunStartCommand(server.StartCommands, true)

		if eoc {
			defer func() {
				server.cleanupCommands()
				os.Exit(0)
			}()
		}

		if err != nil {
			log.Errorf("start command failed: %v", err)
		}
	}()

	return nil
}

func (server *Server) initJaegerTracing() error {
	// if enabled, initialize tracing (Jaeger/OpenTracing)
	if jc := server.JaegerConfig; jc != nil && jc.Enable {
		if cfg, err := jaegercfg.FromEnv(); err == nil {
			server.jaegerCfg = cfg
		} else {
			return fmt.Errorf("config: %v", err)
		}

		if server.jaegerCfg.ServiceName == `` {
			server.jaegerCfg.ServiceName = sliceutil.OrString(jc.ServiceName, `diecast`)
		}

		if server.jaegerCfg.Sampler == nil {
			server.jaegerCfg.Sampler = new(jaegercfg.SamplerConfig)
		}

		if server.jaegerCfg.Reporter == nil {
			server.jaegerCfg.Reporter = new(jaegercfg.ReporterConfig)
		}

		if r := server.jaegerCfg.Reporter; r != nil {
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

		if s := server.jaegerCfg.Sampler; s != nil {
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
				server.jaegerCfg.Reporter.BufferFlushInterval = bfi
			} else {
				return fmt.Errorf("invalid flush interval (minimum: 1ms)")
			}
		}

		if len(jc.Tags) > 0 {
			for k, v := range jc.Tags {
				server.jaegerCfg.Tags = append(server.jaegerCfg.Tags, opentracing.Tag{
					Key:   k,
					Value: v,
				})
			}
		}

		server.jaegerCfg.Tags = append(server.jaegerCfg.Tags, opentracing.Tag{
			Key:   `diecast-version`,
			Value: ApplicationVersion,
		})

		if ott, otc, err := server.jaegerCfg.NewTracer(); err == nil {
			server.opentrace = ott
			server.otcloser = otc

			opentracing.SetGlobalTracer(server.opentrace)

			var logline string

			if v := server.jaegerCfg.Reporter.CollectorEndpoint; v != `` {
				logline = fmt.Sprintf("collector at %s", v)
			} else if v := server.jaegerCfg.Reporter.LocalAgentHostPort; v != `` {
				logline = fmt.Sprintf("agent at %s", v)
			}

			if logline != `` {
				log.Debugf("trace: Jaeger tracing enabled: service=%s send to %s", server.jaegerCfg.ServiceName, logline)

				if len(server.jaegerCfg.Tags) > 0 {
					log.Debugf("trace: global tags:")
					for _, tag := range server.jaegerCfg.Tags {
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
func (server *Server) RenderPath(w io.Writer, path string) error {
	path = `/` + strings.TrimPrefix(path, `/`)

	var rw = httptest.NewRecorder()
	var req = httptest.NewRequest(http.MethodGet, path, nil)
	server.ServeHTTP(rw, req)

	if !rw.Flushed {
		rw.Flush()
	}

	if res := rw.Result(); res.StatusCode < 400 {
		_, err := io.Copy(w, res.Body)
		return err
	} else {
		errbody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("render failed: %v", sliceutil.Or(string(errbody), res.Status))
	}
}

// Perform a single request to the server and return an http.Response.
func (server *Server) GetResponse(method string, path string, body io.Reader, params map[string]any, headers map[string]any) *http.Response {
	path = `/` + strings.TrimPrefix(path, `/`)

	var rw = httptest.NewRecorder()
	var req = httptest.NewRequest(method, path, body)

	for k, v := range params {
		httputil.SetQ(req.URL, k, v)
	}

	for k, v := range headers {
		req.Header.Set(k, typeutil.String(v))
	}

	server.ServeHTTP(rw, req)

	if !rw.Flushed {
		rw.Flush()
	}

	return rw.Result()
}

// Return a URL string that can be used to perform requests from the local machine.
func (server *Server) LocalURL() string {
	return server.bestInternalLoopbackUrl(nil)
}

// Start a long-running webserver.  If provided, the functions provided will be run in parallel
// after the server has started.  If any of them return a non-nil error, the server will stop and
// this method will return that error.
func (server *Server) Serve(workers ...ServeFunc) error {
	var serveable Serveable
	var useTLS bool
	var useUDP bool
	var useSocket string
	var servechan = make(chan error)

	// fire off some goroutines for the prestart and start commands (if configured)
	if err := server.prestart(); err != nil {
		return err
	}

	var srv = &http.Server{
		Handler: server,
	}

	// work out if we're starting a UNIX socket server
	if addr := server.Address; strings.HasPrefix(addr, `unix:`) {
		useSocket = strings.TrimPrefix(addr, `unix:`)

		if useSocket == `` {
			useSocket = `diecast.` + typeutil.String(os.Getpid()) + `.sock`
		}
	} else {
		srv.Addr = addr
	}

	// setup TLSConfig
	if ssl := server.TLS; ssl != nil && ssl.Enable {
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
					"invalid value %q for 'ssl_client_certs': must be one of %q, %q, %q, %q",
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
	switch strings.ToLower(server.Protocol) {
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
		return fmt.Errorf("unknown protocol %q", server.Protocol)
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
					servechan <- serveable.ServeTLS(listener, server.TLS.CertFile, server.TLS.KeyFile)
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
				servechan <- serveable.ListenAndServeTLS(server.TLS.CertFile, server.TLS.KeyFile)
			} else {
				servechan <- serveable.ListenAndServe()
			}
		}()
	}

	if len(workers) > 0 {
		for _, worker := range workers {
			go func(w ServeFunc) {
				servechan <- w(server)
			}(worker)
		}
	}

	return <-servechan
}

func (server *Server) ListenAndServe(address string) error {
	server.Address = address
	return server.Serve()
}

func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// make sure we close the body no matter what
	if req.Body != nil {
		defer req.Body.Close()
	}

	// set Connection header
	if !server.PreserveConnections {
		w.Header().Set(`Connection`, `close`)
	}

	// initialize if necessary. an error here is severe and panics
	if !server.initialized {
		if err := server.Initialize(); err != nil {
			panic(err.Error())
		}
	}

	// perform rate limiting check
	if rl := server.RateLimit; rl != nil && rl.Enable && rl.Limit != `` {
		if server.rateLimiter == nil {
			var lim = ratelimit.CreateLimit(rl.Limit)
			server.rateLimiter = &lim
		}

		if err := server.rateLimiter.Hit(rl.KeyFor(req)); err != nil {
			var didPenalty bool

			// impose sleep penalty if specified
			if penalty := rl.Penalty; penalty != `` {
				if pd := typeutil.Duration(penalty); pd > 0 {
					time.Sleep(pd)
					didPenalty = true
				}
			}

			if !didPenalty {
				server.respondError(w, req, err, http.StatusTooManyRequests)
				return
			}
		}
	}

	// setup a ResponseWriter interceptor that catches status code and bytes written
	// but passes through the Body without buffering it (like httptest.ResponseRecorder does)
	var interceptor = intercept(w)
	httputil.RequestSetValue(req, ContextResponseKey, interceptor)

	// process the before stack
	for i, before := range server.BeforeHandlers {
		if proceed := before(interceptor, req); !proceed {
			log.Debugf(
				"[%s] processing halted by middleware %d (msg: %v)",
				reqid(req),
				i,
				httputil.RequestGetValue(req, ContextErrorKey),
			)

			server.respondError(interceptor, req, fmt.Errorf("middleware halted request"), http.StatusInternalServerError)
			return
		}
	}

	// finally, pass the request on to the ServeMux router
	server.mux.ServeHTTP(interceptor, req)

	// process the middlewares
	for _, after := range server.AfterHandlers {
		after(interceptor, req)
	}
}

// return whether the request path matches any of the configured TemplatePatterns.
func (server *Server) shouldApplyTemplate(requestPath string) bool {
	var baseName = filepath.Base(requestPath)

	for _, pattern := range server.TemplatePatterns {
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
func (server *Server) shouldAutocompress(requestPath string) bool {
	var baseName = filepath.Base(requestPath)

	for _, pattern := range server.AutocompressPatterns {
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
func (server *Server) shouldApplyLayout(requestPath string) bool {
	var baseName = filepath.Base(requestPath)

	for _, pattern := range server.AutolayoutPatterns {
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
func (server *Server) applyTemplate(
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
		if err := server.appendIncludes(&fragments, header); err != nil {
			return err
		}
	}

	var earlyData = server.requestToEvalData(req, header)

	// get a reference to a set of standard functions that won't have a scope yet
	var earlyFuncs = server.GetTemplateFunctions(earlyData, header)

	// only process layouts if we're supposed to
	if server.EnableLayouts && !forceSkipLayout && server.shouldApplyLayout(requestPath) {
		// files starting with "_" are partials and should not have layouts applied
		if !strings.HasPrefix(path.Base(requestPath), `_`) {
			// if no layouts were explicitly specified, and a layout named "default" exists, add it to the list
			if len(layouts) == 0 {
				if _, err := server.LoadLayout(`default`); err == nil {
					layouts = append(layouts, `default`)
				}
			}

			if len(layouts) > 0 {
				for _, layoutName := range layouts {
					if layoutName, err := EvalInline(layoutName, nil, earlyFuncs); err == nil {
						if layoutFile, err := server.LoadLayout(layoutName); err == nil {
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
	var finalHeader = fragments.Header(server)

	// add all includes
	if err := server.appendIncludes(&fragments, &finalHeader); err != nil {
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

	if funcs, data, err := server.GetTemplateData(req, &finalHeader); err == nil {
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

				var usePath string

				if c, err := EvalInline(swcase.UsePath, data, funcs); err == nil {
					usePath = c
				} else {
					return err
				}

				if swcase.Break {
					break SwitchCaseLoop

				} else if swcase.Fallthrough {
					fallingThrough = true
					continue SwitchCaseLoop

				} else if redir := swcase.Redirect; redir != nil {
					finalHeader.Redirect = redir
					break SwitchCaseLoop

				} else if swTemplate, err := server.fs.Open(usePath); err == nil {
					if swHeader, swData, err := SplitTemplateHeaderContent(swTemplate); err == nil {
						if fh, err := finalHeader.Merge(swHeader); err == nil {
							log.Debugf("[%s] Switch case %d matched, switching to template %v", reqid(req), i, usePath)
							// httputil.RequestSetValue(req, SwitchCaseKey, usePath)

							return server.applyTemplate(
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
			if r, ok := GetRendererForFilename(requestPath, server); ok {
				postTemplateRenderer = r
			}
		default:
			if r, err := GetRenderer(finalHeader.Renderer, server); err == nil {
				postTemplateRenderer = r
			} else {
				return err
			}
		}

		// evaluate and render the template first
		if baseRenderer, err := GetRenderer(``, server); err == nil {
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
					renderOpts.Input = io.NopCloser(bytes.NewBuffer(templateData))
				}

				if err == nil {
					// run the final template render and return
					log.Debugf("[%s] renderer: %T", reqid(req), postTemplateRenderer)

					postTemplateRenderer.SetPrewriteFunc(func(r *http.Request) {
						reqtime(r, `tpl`, time.Since(start))
						writeRequestTimerHeaders(server, w, r)
					})

					return postTemplateRenderer.Render(w, req, renderOpts)
				} else {
					return err
				}
			} else {
				// just render the base template directly to the response and return

				baseRenderer.SetPrewriteFunc(func(r *http.Request) {
					reqtime(r, `tpl`, time.Since(start))
					writeRequestTimerHeaders(server, w, r)
				})

				return baseRenderer.Render(w, req, renderOpts)
			}
		} else {
			return err
		}
	} else if redir, ok := err.(RedirectTo); ok {
		log.Debugf("[%s] Performing 307 Temporary Redirect to %v due to binding response handler.", reqid(req), redir)
		writeRequestTimerHeaders(server, w, req)
		http.Redirect(w, req, redir.Error(), http.StatusTemporaryRedirect)
		return nil
	} else {
		return err
	}
}

// Retrieves the set of standard template functions, as well as functions for working
// with data in the current request.
func (server *Server) GetTemplateFunctions(data map[string]any, header *TemplateHeader) FuncMap {
	var funcs = make(FuncMap)

	for k, v := range GetStandardFunctions(server) {
		funcs[k] = v
	}

	if server.AdditionalFunctions != nil {
		for k, v := range server.AdditionalFunctions {
			funcs[k] = v
		}
	}

	// fn payload: Return the body supplied with the request used to generate the current view.
	funcs[`payload`] = func(key ...string) any {
		if len(key) == 0 {
			return data
		} else {
			return maputil.DeepGet(data, strings.Split(key[0], `.`), nil)
		}
	}

	// fn querystrings: Return a map of all of the query string parameters in the current URL.
	funcs[`querystrings`] = func() map[string]any {
		if reqinfo, ok := data[`_request`].(*RequestInfo); ok {
			return reqinfo.URL.Query
		}

		return make(map[string]any)
	}

	// fn cookie: return a cookie value
	funcs[`cookie`] = func(key any, fallbacks ...any) any {
		if len(fallbacks) == 0 {
			fallbacks = []any{``}
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
	funcs[`qs`] = func(key any, fallbacks ...any) any {
		if len(fallbacks) == 0 {
			fallbacks = []any{``}
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
	funcs[`headers`] = func(key string, fallbacks ...any) string {
		if len(fallbacks) == 0 {
			fallbacks = []any{``}
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
	funcs[`param`] = func(nameOrIndex any, fallbacks ...any) any {
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
	funcs[`var`] = func(name string, vI ...any) any {
		var value any

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
	funcs[`varset`] = func(name string, key string, vI ...any) any {
		var value any
		var path = makeVarKey(name)

		switch len(vI) {
		case 0:
			value = make(map[string]any)
		case 1:
			value = vI[0]
		default:
			value = vI
		}

		maputil.DeepSet(data, append(path, strings.Split(key, `.`)...), value)
		return ``
	}

	// fn push: Append to variable *name* to *value*.
	funcs[`push`] = func(name string, vI ...any) any {
		var values []any
		var key = makeVarKey(name)

		if existing := maputil.DeepGet(data, key); existing != nil {
			values = append(values, sliceutil.Sliceify(existing)...)
		}

		values = append(values, vI...)
		maputil.DeepSet(data, key, values)

		return ``
	}

	// fn pop: Remove the last item from *name* and return it.
	funcs[`pop`] = func(name string) any {
		var out any
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
	funcs[`increment`] = func(name string, incr ...any) any {
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
	funcs[`incrementByValue`] = func(name string, value any, incr ...any) any {
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
		if data, err := readFromFS(server.fs, filename); err == nil {
			return string(data), nil
		} else {
			return ``, err
		}
	}

	// read a file from the serving path and parse it as a template, returning the output.
	funcs[`render`] = func(filename string, overrides ...map[string]any) (string, error) {
		if tpl, err := readFromFS(server.fs, filename); err == nil {
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
		if server.Locale != `` {
			if tag, err := language.Parse(server.Locale); err == nil {
				locales = append(locales, tag.String())
				locales = append(locales, i18nTagBase(tag))
			} else {
				log.Warningf("i18n: invalid global locale %q", server.Locale)
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

		for _, translations := range []map[string]any{
			header.Translations,
			server.Translations,
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
		if len(server.JWT) > 0 {
			if cfg, ok := server.JWT[jwtConfigName]; ok && cfg != nil {
				return cfg.Issue(data, funcs)
			}
		}

		return ``, fmt.Errorf("JWT configuration %q not found", jwtConfigName)
	}

	// fn jwtSign: generates a signature for the given input.  If the input is a map or array, it will be
	// JSON-encoded before signing.
	funcs[`jwtSign`] = func(jwtConfigName string, input any) (string, error) {
		if len(server.JWT) > 0 {
			if cfg, ok := server.JWT[jwtConfigName]; ok && cfg != nil {
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
		if len(server.JWT) > 0 {
			if cfg, ok := server.JWT[jwtConfigName]; ok && cfg != nil {
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

func (server *Server) LoadLayout(name string) (io.Reader, error) {
	return server.fs.Open(fmt.Sprintf("%s/%s.html", server.LayoutPath, name))
}

func (server *Server) ToTemplateName(requestPath string) string {
	return requestPath
}

// gets a FuncMap and data usable in templates and error pages alike, before bindings are evaluated.
func (server *Server) getPreBindingData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]any) {
	var data = server.requestToEvalData(req, header)
	var funcs = server.GetTemplateFunctions(data, header)

	data[`vars`] = make(map[string]any)

	var publicMountDetails = make([]map[string]any, 0)

	for _, mount := range server.MountConfigs {
		publicMountDetails = append(publicMountDetails, map[string]any{
			`from`: mount.Mount,
			`to`:   mount.To,
		})
	}

	data[`diecast`] = map[string]any{
		`binding_prefix`:    server.BindingPrefix,
		`route_prefix`:      server.rp(),
		`template_patterns`: server.TemplatePatterns,
		`try_local_first`:   server.TryLocalFirst,
		`index_file`:        server.IndexFile,
		`verify_file`:       server.VerifyFile,
		`mounts`:            publicMountDetails,
	}

	server.evalPageData(false, req, header, funcs, data)

	return funcs, data
}

func (server *Server) evalPageData(final bool, _ *http.Request, header *TemplateHeader, funcs FuncMap, data map[string]any) map[string]any {
	// Evaluate "page" data: this data is templatized, but does not have access
	//                       to the output of bindings
	// ---------------------------------------------------------------------------------------------
	var pageData = make(map[string]any)

	if header != nil {
		var applyPageFn = func(value any, path []string, isLeaf bool) error {

			if isLeaf {
				switch vtyped := value.(type) {
				case string:
					if v, err := EvalInline(vtyped, data, funcs); err == nil {
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
		maputil.Walk(server.DefaultPageObject, applyPageFn)

		// then pepper in whatever values came from the aggregated headers from
		// the layout, includes, and target template
		maputil.Walk(header.Page, applyPageFn)

		// if there were override items specified (e.g.: via the command line), add them now
		maputil.Walk(server.OverridePageObject, applyPageFn)
	}

	data[`page`] = pageData
	data[`p`] = pageData

	return pageData
}

func (server *Server) updateBindings(data map[string]any, with map[string]any) {
	data[`bindings`] = with
	data[`b`] = with
}

func (server *Server) GetTemplateData(req *http.Request, header *TemplateHeader) (FuncMap, map[string]any, error) {
	var funcs, data = server.getPreBindingData(req, header)

	// Evaluate "bindings": Bindings have access to $.page, and each subsequent binding has access
	//                      to all binding output that preceded it.  This allows bindings to be
	//                      pipelined, using the output of one request as the input of the next.
	// ---------------------------------------------------------------------------------------------
	var bindings = maputil.M(&server.sharedBindingData).MapNative()
	var bindingsToEval = make([]Binding, 0)

	// only use top-level bindings that
	bindingsToEval = append(bindingsToEval, server.Bindings.perRequestBindings()...)

	if header != nil {
		bindingsToEval = append(bindingsToEval, header.Bindings...)
	}

	for i, binding := range bindingsToEval {
		if strings.TrimSpace(binding.Name) == `` {
			binding.Name = fmt.Sprintf("binding%d", i)
		}

		binding.server = server

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
			var results = make([]map[string]any, 0)
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
					binding.Params = make(map[string]any)
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
					server.updateBindings(data, bindings)
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

				server.updateBindings(data, bindings)
				page++
			}

			bindings[binding.Name] = results
			server.updateBindings(data, bindings)

		} else if binding.Repeat == `` {
			bindings[binding.Name] = binding.Fallback
			server.updateBindings(data, bindings)

			v, err := binding.tracedEvaluate(req, header, data, funcs)

			if err == nil && v != nil {
				bindings[binding.Name] = v
				server.updateBindings(data, bindings)
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
			var results = make([]any, 0)

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
					server.updateBindings(data, bindings)
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

				server.updateBindings(data, bindings)
			}

		}

		reqtime(req, fmt.Sprintf("binding-%s", binding.Name), time.Since(start))

		// re-evaluate page based on new binding results
		server.evalPageData(false, req, header, funcs, data)
	}

	server.updateBindings(data, bindings)

	// Evaluate "flags" data: this data is templatized, and has access to $.page and $.bindings
	// ---------------------------------------------------------------------------------------------
	if header != nil {
		var flags = make(map[string]bool)

		for name, def := range header.FlagDefs {
			switch def := def.(type) {
			case bool:
				flags[name] = def
			default:
				if flag, err := EvalInline(typeutil.String(def), data, funcs); err == nil {
					flags[name] = typeutil.V(flag).Bool()
				} else {
					return nil, nil, fmt.Errorf("flags: %v", err)
				}
			}
		}

		data[`flags`] = flags
	}

	// the final pass on page; any empty values resulting from this are final
	server.evalPageData(true, req, header, funcs, data)

	return funcs, data, nil
}

func (server *Server) tryAutoindex() (http.File, string, bool) {
	if autoindex, err := server.fs.Open(server.AutoindexTemplate); err == nil {
		return autoindex, `text/html`, true
	} else if autoindex, err := http.FS(embedded).Open(server.AutoindexTemplate); err == nil {
		return autoindex, `text/html`, true
	} else {
		return nil, ``, false
	}
}

// Attempt to resolve the given path into a real file and return that file and mime type.
// Non-existent files, unreadable files, and directories will return an error.
func (server *Server) tryLocalFile(requestPath string, _ *http.Request) (http.File, string, error) {
	// if we got here, try to serve the file from the filesystem
	if file, err := server.fs.Open(requestPath); err == nil {
		if stat, err := file.Stat(); err == nil {
			if !stat.IsDir() {
				if mimetype, err := figureOutMimeType(stat.Name(), file); err == nil {
					return file, mimetype, nil
				} else {
					return file, ``, err
				}
			} else {
				return file, ``, ErrIsDirectory
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
func (server *Server) tryMounts(requestPath string, req *http.Request) (Mount, *MountResponse, error) {
	var body *RequestBody

	if rb := reqbody(req); rb != nil {
		body = rb
	} else {
		return nil, nil, fmt.Errorf("no request body")
	}

	var lastErr error

	// find a mount that has this file
	for _, mount := range server.Mounts {
		// closing the RequestBody resets the reader to the beginning
		body.Close()

		if mount.WillRespondTo(requestPath, req, body) {
			// attempt to open the file entry
			mountResponse, err := mount.OpenWithType(requestPath, req, body)
			lastErr = err

			if err == nil && !mountResponse.IsDir() {
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

func (server *Server) respondError(w http.ResponseWriter, req *http.Request, resErr error, code int) {
	var tmpl = NewTemplate(`error`, HtmlEngine)

	if resErr == nil {
		resErr = fmt.Errorf("unknown Error")
	}

	if c := httputil.RequestGetValue(req, ContextStatusKey).Int(); c > 0 {
		code = int(c)
	}

	for _, filename := range []string{
		fmt.Sprintf("%s/%d.html", server.ErrorsPath, code),
		fmt.Sprintf("%s/%dxx.html", server.ErrorsPath, int(code/100.0)),
		fmt.Sprintf("%s/default.html", server.ErrorsPath),
	} {
		if f, err := server.fs.Open(filename); err == nil {
			funcs, errorData := server.getPreBindingData(req, server.BaseHeader)
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
	if data, err := io.ReadAll(reader); err == nil {
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

func (server *Server) appendIncludes(fragments *FragmentSet, header *TemplateHeader) error {
	if header != nil {
		for name, includePath := range header.Includes {
			if includeFile, err := server.fs.Open(includePath); err == nil {
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

func (server *Server) actionForRequest(req *http.Request) http.HandlerFunc {
	var route = req.URL.Path

	for _, action := range server.Actions {
		var actionPath = filepath.Join(server.rp(), action.Path)

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

func (server *Server) rp() string {
	return strings.TrimSuffix(server.RoutePrefix, `/`)
}

func (server *Server) requestToEvalData(req *http.Request, header *TemplateHeader) map[string]any {
	var rv = map[string]any{}

	var request = RequestInfo{
		Headers: make(map[string]any),
		Cookies: make(map[string]Cookie),
		URL: RequestUrlInfo{
			Query: make(map[string]any),
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
	var env = make(map[string]any)

	for _, pair := range os.Environ() {
		key, value := stringutil.SplitPair(pair, `=`)
		key = envKeyNorm(key)

		if server.mayExposeEnvVar(key) {
			env[key] = stringutil.Autotype(value)
		}
	}

	rv[`env`] = env

	return rv
}

func (server *Server) RunStartCommand(scmds []*StartCommand, waitForCommand bool) (bool, error) {
	for _, scmd := range scmds {
		if cmdline := scmd.Command; cmdline != `` {
			if tokens, err := shellwords.Parse(cmdline); err == nil {
				scmd.cmd = exec.Command(tokens[0], tokens[1:]...)
				scmd.cmd.SysProcAttr = &syscall.SysProcAttr{
					Setpgid: true,
				}

				var env = make(map[string]any)

				for _, pair := range os.Environ() {
					key, value := stringutil.SplitPair(pair, `=`)
					env[key] = value
				}

				for key, value := range scmd.Environment {
					env[key] = value
				}

				env[`DIECAST`] = true
				env[`DIECAST_BIN`] = server.BinPath
				env[`DIECAST_DEBUG`] = server.EnableDebugging
				env[`DIECAST_ADDRESS`] = server.Address
				env[`DIECAST_ROOT`] = server.RootPath
				env[`DIECAST_PATH_LAYOUTS`] = server.LayoutPath
				env[`DIECAST_PATH_ERRORS`] = server.ErrorsPath
				env[`DIECAST_BINDING_PREFIX`] = server.BindingPrefix
				env[`DIECAST_ROUTE_PREFIX`] = server.rp()

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

func (server *Server) mayExposeEnvVar(name string) bool {
	name = envKeyNorm(name)

	for _, f := range server.ExposeEnvVars {
		if glob.MustCompile(envKeyNorm(f)).Match(name) {
			return true
		}
	}

	for _, f := range server.FilterEnvVars {
		if glob.MustCompile(envKeyNorm(f)).Match(name) {
			return false
		}
	}

	return true
}

func (server *Server) cleanupCommands() {
	for _, psc := range server.PrestartCommands {
		if psc.cmd != nil {
			if proc := psc.cmd.Process; proc != nil {
				proc.Kill()
			}
		}
	}

	for _, sc := range server.StartCommands {
		if sc.cmd != nil {
			if proc := sc.cmd.Process; proc != nil {
				proc.Kill()
			}
		}
	}
}

// called by the cleanup middleware to log the completed request according to LogFormat.
func (server *Server) logreq(_ http.ResponseWriter, req *http.Request) {
	if server.Log.Disable {
		return
	}

	if tm := getRequestTimer(req); tm != nil {
		var format = logFormats[server.Log.Format]

		if format == `` {
			if server.Log.Format != `` {
				format = server.Log.Format
			} else {
				return
			}
		}

		if server.logwriter == nil {
			// discard by default, unless some brave configuration below changes this
			server.logwriter = io.Discard

			switch lf := strings.ToLower(server.Log.Destination); lf {
			case ``, `none`, `false`:
				return
			case `-`, `stdout`:
				server.isTerminalOutput = true
				server.logwriter = os.Stdout
			case `stderr`:
				server.isTerminalOutput = true
				server.logwriter = os.Stderr
			case `syslog`:
				log.Warningf("logfile: %q destination is not implemented", lf)
				return
			default:
				if server.Log.Truncate {
					os.Truncate(server.Log.Destination, 0)
				}

				if f, err := os.OpenFile(server.Log.Destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
					server.logwriter = f
				} else {
					log.Warningf("logfile: failed to open logfile: %v", err)
					return
				}
			}
		}

		var interceptor = reqres(req)
		rh, rp := stringutil.SplitPair(req.RemoteAddr, `:`)
		var code = typeutil.String(interceptor.code)

		if server.isTerminalOutput && server.Log.Colorize {
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

		var logContext = maputil.M(map[string]any{
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

		logContext.Fprintf(server.logwriter, format)
	} else {
		bugWarning()
	}
}

func (server *Server) bindingTimeout() time.Duration {
	if t := typeutil.Duration(server.BindingTimeout); t > 0 {
		return t
	} else {
		return DefaultBindingTimeout
	}
}

func (server *Server) bestInternalLoopbackUrl(req *http.Request) string {
	if server.BindingPrefix != `` {
		return server.BindingPrefix
	}

	var proto string

	if server.TLS != nil && server.TLS.Enable {
		proto = `https`
	} else {
		proto = `http`
	}

	if strings.HasPrefix(server.Address, `unix:`) {
		var path = server.Address

		path = strings.TrimPrefix(path, `unix:`)
		path = strings.ReplaceAll(path, `/`, weirdPathsInHostnamesPlaceholder)

		return proto + `+unix://` + path

	} else if h, p, err := net.SplitHostPort(server.Address); err == nil {
		switch h {
		case `0.0.0.0`, `::/0`, `[::/0]`:
			return proto + `://localhost:` + p
		}
	}

	if req != nil && req.Host != `` {
		return proto + `://` + req.Host
	} else {
		return proto + `://` + server.Address
	}
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

	data, err := io.ReadAll(req.Body)
	req.Body.Close()

	if err == nil {
		req.Body = io.NopCloser(bytes.NewBuffer(data))
		return strings.Join(request, "\r\n") + "\r\n\r\n" + string(data)
	} else {
		request = append(request, fmt.Sprintf("\nFAILED to read body: %v", err))
	}

	return strings.Join(request, "\n")
}
