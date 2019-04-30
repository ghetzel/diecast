package diecast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghodss/yaml"
)

type BindingErrorAction string

const (
	ActionSummarize BindingErrorAction = `summarize`
	ActionPrint                        = `print`
	ActionContinue                     = `continue`
	ActionBreak                        = `break`
	ActionIgnore                       = `ignore`
)

var BindingClient = &http.Client{
	Timeout: 60 * time.Second,
}

var AllowInsecureLoopbackBindings bool
var DefaultParamJoiner = `;`

type Binding struct {
	Name               string                     `json:"name,omitempty"`
	Restrict           []string                   `json:"restrict,omitempty"`
	OnlyIfExpr         string                     `json:"only_if,omitempty"`
	NotIfExpr          string                     `json:"not_if,omitempty"`
	Method             string                     `json:"method,omitempty"`
	Resource           string                     `json:"resource,omitempty"`
	Timeout            interface{}                `json:"timeout,omitempty"`
	Insecure           bool                       `json:"insecure,omitempty"`
	ParamJoiner        string                     `json:"param_joiner,omitempty"`
	Params             map[string]interface{}     `json:"params,omitempty"`
	Headers            map[string]string          `json:"headers,omitempty"`
	BodyParams         map[string]interface{}     `json:"body,omitempty"`
	RawBody            string                     `json:"rawbody,omitempty"`
	Formatter          string                     `json:"formatter,omitempty"`
	Parser             string                     `json:"parser,omitempty"`
	NoTemplate         bool                       `json:"no_template,omitempty"`
	Optional           bool                       `json:"optional,omitempty"`
	Fallback           interface{}                `json:"fallback,omitempty"`
	OnError            BindingErrorAction         `json:"on_error,omitempty"`
	IfStatus           map[int]BindingErrorAction `json:"if_status,omitempty"`
	Repeat             string                     `json:"repeat,omitempty"`
	SkipInheritHeaders bool                       `json:"skip_inherit_headers,omitempty"`
	DisableCache       bool                       `json:"disable_cache,omitempty"`
	server             *Server
}

func (self *Binding) ShouldEvaluate(req *http.Request) bool {
	if self.Restrict == nil {
		return true
	} else {
		for _, restrict := range self.Restrict {
			if rx, err := regexp.Compile(restrict); err == nil {
				if rx.MatchString(req.URL.Path) {
					return true
				}
			}
		}
	}

	return false
}

func (self *Binding) Evaluate(req *http.Request, header *TemplateHeader, data map[string]interface{}, funcs FuncMap) (interface{}, error) {
	id := reqid(req)
	log.Debugf("[%s] Evaluating binding %q", id, self.Name)

	if req.Header.Get(`X-Diecast-Binding`) == self.Name {
		return nil, fmt.Errorf("Loop detected")
	}

	method := strings.ToUpper(self.Method)
	uri := EvalInline(self.Resource, data, funcs)

	// bindings may specify that a request should be made to the currently server address by
	// prefixing the URL path with a colon (":") or slash ("/").
	//
	if strings.HasPrefix(uri, `:`) || strings.HasPrefix(uri, `/`) {
		var prefix string

		if self.server.BindingPrefix != `` {
			prefix = self.server.BindingPrefix
		} else {
			prefix = fmt.Sprintf("http://%s", req.Host)
		}

		prefix = strings.TrimSuffix(prefix, `/`)
		uri = strings.TrimPrefix(uri, `:`)
		uri = strings.TrimPrefix(uri, `/`)

		uri = fmt.Sprintf("%s/%s", prefix, uri)

		// allows bindings referencing the local server to avoid TLS cert verification
		// because the prefix is often `localhost:port`, which probably won't verify anyway.
		if AllowInsecureLoopbackBindings {
			self.Insecure = true
		}
	}

	if !self.NoTemplate {
		if self.OnlyIfExpr != `` {
			if v := EvalInline(self.OnlyIfExpr, data, funcs); typeutil.IsEmpty(v) || stringutil.IsBooleanFalse(v) {
				self.Optional = true
				return nil, fmt.Errorf("[%s] Binding %q not being evaluated because only_if expression was false", id, self.Name)
			}
		}

		if self.NotIfExpr != `` {
			if v := EvalInline(self.NotIfExpr, data, funcs); !typeutil.IsEmpty(v) && !stringutil.IsBooleanFalse(v) {
				self.Optional = true
				return nil, fmt.Errorf("[%s] Binding %q not being evaluated because not_if expression was truthy", id, self.Name)
			}
		}
	}

	if reqUrl, err := url.Parse(uri); err == nil {
		var protocol Protocol

		switch reqUrl.Scheme {
		case `http`, `https`, ``:
			protocol = new(HttpProtocol)
		case `redis`:
			protocol = new(RedisProtocol)
		default:
			return nil, fmt.Errorf("Cannot evaluate binding %v: invalid protocol scheme %q", self.Name, reqUrl.Scheme)
		}

		log.Debugf("[%s]  binding %q: protocol=%T uri=%v", id, self.Name, protocol, uri)
		log.Infof("[%s] Binding: > %s %+v ? %s", id, strings.ToUpper(sliceutil.OrString(method, `get`)), reqUrl.String(), reqUrl.RawQuery)

		if response, err := protocol.Retrieve(&ProtocolRequest{
			Verb:          method,
			URL:           reqUrl,
			Binding:       self,
			Request:       req,
			Header:        header,
			TemplateData:  data,
			TemplateFuncs: funcs,
		}); err == nil {
			defer response.Close()

			onError := BindingErrorAction(EvalInline(string(self.OnError), data, funcs))

			// handle per-http-status response handlers
			if len(self.IfStatus) > 0 && response.StatusCode > 0 {
				// get the action for this code
				if statusAction, ok := self.IfStatus[response.StatusCode]; ok {
					statusAction = BindingErrorAction(EvalInline(string(statusAction), data, funcs))

					switch statusAction {
					case ActionIgnore:
						onError = ActionIgnore
					default:
						redirect := string(statusAction)

						if !self.NoTemplate {
							redirect = EvalInline(redirect, data, funcs)
						}

						// if a url or path was specified, redirect the parent request to it
						if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
							return nil, RedirectTo(redirect)
						} else {
							return nil, fmt.Errorf("[%s] Invalid status action '%v'", id, redirect)
						}
					}
				}
			}

			if data, err := ioutil.ReadAll(response); err == nil {
				if response.StatusCode >= 400 {
					switch onError {
					case ActionPrint:
						return nil, fmt.Errorf("%v", string(data[:]))
					case ActionIgnore:
						break
					default:
						redirect := string(onError)

						// if a url or path was specified, redirect the parent request to it
						if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
							return nil, RedirectTo(redirect)
						} else {
							return nil, fmt.Errorf(
								"[%s] Request %s %v failed: %v",
								id,
								method,
								reqUrl,
								response.StatusCode,
							)
						}
					}
				}

				// only do response body processing if there is data to process
				if len(data) > 0 {
					if self.Parser == `` {
						switch response.MimeType {
						case `application/json`:
							self.Parser = `json`
						case `application/x-yaml`, `application/yaml`, `text/yaml`:
							self.Parser = `yaml`
						case `text/html`:
							self.Parser = `html`
						case `text/xml`:
							self.Parser = `xml`
						}
					}

					switch self.Parser {
					case `json`, ``:
						// if the parser is unset, and the response type is NOT application/json, then
						// just read the response as plain text and return it.
						//
						// If you're certain the response actually is JSON, then explicitly set Parser==`json`
						//
						if self.Parser == `` && response.MimeType != `application/json` {
							return string(data), nil
						} else {
							var rv interface{}

							if err := json.Unmarshal(data, &rv); err == nil {
								return rv, nil
							} else {
								return nil, err
							}
						}

					case `yaml`:
						var rv interface{}
						if err := yaml.Unmarshal(data, &rv); err == nil {
							return rv, nil
						} else {
							return nil, err
						}

					case `html`:
						return goquery.NewDocumentFromReader(bytes.NewBuffer(data))

					case `tsv`:
						return xsvToArray(data, '\t')

					case `csv`:
						return xsvToArray(data, ',')

					case `xml`:
						return xmlToMap(data)

					case `text`:
						return string(data), nil

					case `raw`:
						return template.HTML(string(data)), nil

					default:
						return nil, fmt.Errorf("[%s] Unknown response parser %q", id, self.Parser)
					}
				} else {
					return nil, nil
				}
			} else {
				return nil, fmt.Errorf("[%s] Failed to read response body: %v", id, err)
			}
		} else {
			return nil, fmt.Errorf("[%s] HTTP %v", id, err)
		}
	} else {
		return nil, fmt.Errorf("[%s] url: %v", id, err)
	}
}

func EvalInline(input string, data map[string]interface{}, funcs FuncMap) string {
	tmpl := NewTemplate(`inline`, HtmlEngine)
	tmpl.Funcs(funcs)

	if err := tmpl.ParseString(input); err == nil {
		output := bytes.NewBuffer(nil)

		if err := tmpl.Render(output, data, ``); err == nil {
			// since this data may have been entity escaped by html/template, unescape it here
			return html.UnescapeString(output.String())
		} else {
			panic(fmt.Sprintf("error evaluating %q: %v", input, err))
		}
	}

	return input
}
