package diecast

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/go-shiori/go-readability"
	base58 "github.com/jbenet/go-base58"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
)

func loadStandardFunctionsCodecs(funcs FuncMap, server *Server) funcGroup {
	return funcGroup{
		Name:        `Encoding and Decoding`,
		Description: `For encoding typed data and data structures into well-known formats like JSON, CSV, and TSV.`,
		Functions: []funcDef{
			{
				Name:    `jsonify`,
				Summary: `Encode the given argument as a JSON document.`,
				Arguments: []funcArg{
					{
						Name:        `data`,
						Type:        `any`,
						Description: `The data to encode as JSON.`,
					}, {
						Name:        `indent`,
						Type:        `string`,
						Optional:    true,
						Default:     `  `,
						Description: `The string to indent successive tiers in the document hierarchy with.`,
					},
				},
				Function: func(value interface{}, indent ...string) (string, error) {
					indentString := `  `

					if len(indent) > 0 {
						indentString = indent[0]
					}

					data, err := json.MarshalIndent(value, ``, indentString)
					return string(data[:]), err
				},
			}, {
				// fn markdown: Render the given Markdown string *value* as sanitized HTML.
				Name:    `markdown`,
				Summary: `Parse the given string as a Markdown document and return it represented as HTML.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The full text of the Markdown to parse`,
					}, {
						Name:        `extensions`,
						Type:        `string(s)`,
						Description: `A list of zero of more Markdown extensions to enable when rendering the HTML.`,
						Valid: []funcArg{
							{
								Name:        `no-intra-emphasis`,
								Description: ``,
							}, {
								Name:        `tables`,
								Description: ``,
							}, {
								Name:        `fenced-code`,
								Description: ``,
							}, {
								Name:        `autolink`,
								Description: ``,
							}, {
								Name:        `strikethrough`,
								Description: ``,
							}, {
								Name:        `lax-html-blocks`,
								Description: ``,
							}, {
								Name:        `space-headings`,
								Description: ``,
							}, {
								Name:        `hard-line-break`,
								Description: ``,
							}, {
								Name:        `tab-size-eight`,
								Description: ``,
							}, {
								Name:        `footnotes`,
								Description: ``,
							}, {
								Name:        `no-empty-line-before-block`,
								Description: ``,
							}, {
								Name:        `heading-ids`,
								Description: ``,
							}, {
								Name:        `titleblock`,
								Description: ``,
							}, {
								Name:        `auto-heading-ids`,
								Description: ``,
							}, {
								Name:        `backslash-line-break`,
								Description: ``,
							}, {
								Name:        `definition-lists`,
								Description: ``,
							}, {
								Name:        `common`,
								Description: ``,
							},
						},
						Variadic: true,
					},
				},
				Function: func(value interface{}, extensions ...string) (template.HTML, error) {
					input := typeutil.String(value)
					output := blackfriday.Run(
						[]byte(input),
						blackfriday.WithExtensions(toMarkdownExt(extensions...)),
					)
					output = bluemonday.UGCPolicy().SanitizeBytes(output)

					return template.HTML(output), nil

					// if doc, err := htmldoc(string(output)); err == nil {
					// 	if contents, err := doc.Find(`body`).Html(); err == nil {
					// 		return template.HTML(contents), nil
					// 	} else {
					// 		return ``, err
					// 	}
					// } else {
					// 	return ``, err
					// }
				},
			}, {
				Name:    `csv`,
				Summary: `Encode the given data as a comma-separated values document.`,
				Arguments: []funcArg{
					{
						Name:        `columns`,
						Type:        `array[string]`,
						Description: `An array of values that represent the column names of the table being created.`,
					}, {
						Name:        `rows`,
						Type:        `array[array[string]], array[object]`,
						Description: `An array of values that represent the column names of the table being created.`,
					},
				},
				Function: func(columns []interface{}, rows []interface{}) (string, error) {
					return delimited(',', columns, rows)
				},
			}, {
				Name:    `tsv`,
				Summary: `Encode the given data as a tab-separated values document.`,
				Arguments: []funcArg{
					{
						Name:        `columns`,
						Type:        `array[string]`,
						Description: `An array of values that represent the column names of the table being created.`,
					}, {
						Name:        `rows`,
						Type:        `array[array[string]], array[object]`,
						Description: `An array of values that represent the column names of the table being created.`,
					},
				},
				Function: func(columns []interface{}, rows []interface{}) (string, error) {
					return delimited('\t', columns, rows)
				},
			}, {
				Name: `unsafe`,
				Summary: `Return an unescaped raw HTML segment for direct inclusion in the rendered template output.` +
					`This function bypasses the built-in HTML escaping and sanitization security features, and you ` +
					`almost certainly want to use [sanitize](#fn-sanitize) instead.  This is a common antipattern that ` +
					`leads to all kinds of security issues from poorly-constrained implementations, so you are forced ` +
					`to acknowledge this by typing "unsafe".`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The raw HTML snippet you sneakily want to sneak past the HTML sanitizer for reasons.`,
					},
				},
				Function: func(value interface{}) (template.HTML, error) {
					switch value.(type) {
					case *goquery.Document:
						if doc, err := value.(*goquery.Document).Html(); err == nil {
							return template.HTML(doc), nil
						} else {
							return ``, err
						}
					default:
						return template.HTML(typeutil.String(value)), nil
					}
				},
			}, {
				Name: `sanitize`,
				Summary: `Takes a raw HTML string and santizes it, removing attributes and elements that can be used ` +
					`to evaluate scripts, but leaving the rest. Useful for preparing user-generated HTML for display.`,
				Arguments: []funcArg{
					{
						Name:        `value`,
						Type:        `string, HTML document object`,
						Description: `The document to sanitize.`,
					},
				},
				Function: func(value interface{}) (template.HTML, error) {
					var document string

					switch value.(type) {
					case *goquery.Document:
						if doc, err := value.(*goquery.Document).Html(); err == nil {
							document = doc
						} else {
							return ``, err
						}
					default:
						document = typeutil.String(value)
					}

					return template.HTML(bluemonday.UGCPolicy().Sanitize(document)), nil
				},
			}, {
				Name:    `readabilize`,
				Summary: `Takes a raw HTML string and a readable version out of it.`,
				Arguments: []funcArg{
					{
						Name:        `value`,
						Type:        `string, HTML document object`,
						Description: `The document to sanitize.`,
					},
				},
				Function: func(url string) (template.HTML, error) {
					if article, err := readability.FromURL(url, 10*time.Second); err == nil {
						buf := bytes.NewBuffer(nil)

						if err := html.Render(buf, article.Node); err == nil {
							return template.HTML(buf.String()), nil
						} else {
							return ``, err
						}
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlencode`,
				Summary: `Encode a given string so it can be safely placed inside a URL query.`,
				Arguments: []funcArg{
					{
						Name:        `string`,
						Type:        `string`,
						Description: `The string to encode.`,
					},
				},
				Function: func(value string) string {
					return url.QueryEscape(value)
				},
			}, {
				Name:    `urldecode`,
				Summary: `Decode a URL-encoded string.`,
				Arguments: []funcArg{
					{
						Name:        `encoded`,
						Type:        `string`,
						Description: `The string to decode.`,
					},
				},
				Function: func(value string) (string, error) {
					return url.QueryUnescape(value)
				},
			}, {
				Name:    `urlPathEncode`,
				Summary: `Encode a given string so it can be safely placed inside a URL path segment.`,
				Arguments: []funcArg{
					{
						Name:        `string`,
						Type:        `string`,
						Description: `The string to encode.`,
					},
				},
				Function: func(value string) string {
					return url.PathEscape(value)
				},
			}, {
				Name:    `urlPathDecode`,
				Summary: `Decode a URL-encoded path.`,
				Arguments: []funcArg{
					{
						Name:        `encoded`,
						Type:        `string`,
						Description: `The string to decode.`,
					},
				},
				Function: func(value string) (string, error) {
					return url.PathUnescape(value)
				},
			}, {
				Name:    `url`,
				Summary: `Builds a URL with querystrings from the given base URL string and object.`,
				Arguments: []funcArg{
					{
						Name:        `baseurl`,
						Type:        `string`,
						Description: `The URL to modify`,
					}, {
						Name:        `querymap`,
						Type:        `object`,
						Optional:    true,
						Description: `A key-value object of query string values to add to the URL.`,
					},
				},
				Function: func(base string, queries ...map[string]interface{}) (string, error) {
					if u, err := url.Parse(base); err == nil {
						for _, qs := range queries {
							for k, v := range qs {
								httputil.SetQ(u, k, v)
							}
						}

						return u.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlScheme`,
				Summary: `Return the scheme portion of the given URL.`,
				Examples: []funcExample{
					{
						Code:   `urlScheme "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: `https`,
					},
				},
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Function: func(in string) (string, error) {
					if u, err := url.Parse(in); err == nil {
						return u.Scheme, nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlHost`,
				Summary: `Return the host[:port] portion of the given URL.`,
				Examples: []funcExample{
					{
						Code:   `urlHost "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: `example.com:8443`,
					}, {
						Code:   `urlHost "https://example.com/somewhere/else/`,
						Return: `example.com`,
					},
				},
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Function: func(in string) (string, error) {
					if u, err := url.Parse(in); err == nil {
						return u.Host, nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlHostname`,
				Summary: `Return the hostname (without port number) portion of the given URL.`,
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `urlHostname "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: `example.com`,
					}, {
						Code:   `urlHostname "https://other.example.com/somewhere/else/`,
						Return: `other.example.com`,
					},
				},
				Function: func(in string) (string, error) {
					if u, err := url.Parse(in); err == nil {
						return u.Hostname(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlPort`,
				Summary: `Return the numeric port number of the given URL.`,
				Examples: []funcExample{
					{
						Code:   `urlPort "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: 8443,
					}, {
						Code:   `urlPort "https://example.com/somewhere/else/`,
						Return: 443,
					},
				},
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Function: func(in string) (int, error) {
					if u, err := url.Parse(in); err == nil {
						if p := u.Port(); p != `` {
							return int(typeutil.Int(p)), nil
						} else {
							return 0, fmt.Errorf("Invalid port number")
						}
					} else {
						return 0, err
					}
				},
			}, {
				Name:    `urlPath`,
				Summary: `Return the path component of the given URL.`,
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `urlPath "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: `/path/to/file.xml`,
					},
				},
				Function: func(in string) (string, error) {
					if u, err := url.Parse(in); err == nil {
						if p := u.Path; strings.HasPrefix(p, `/`) {
							return p, nil
						} else {
							return `/`, nil
						}
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlQueryString`,
				Summary: `Return a querystring value from the given URL.`,
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The querystring value to retrieve.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `urlQueryString "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9" "lang"`,
						Return: `en`,
					},
				},
				Function: func(in string, key string) (string, error) {
					if u, err := url.Parse(in); err == nil {
						return u.Query().Get(key), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `urlQuery`,
				Summary: `Return all querystring values from the given URL.`,
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Examples: []funcExample{
					{
						Code: `urlQuery "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: map[string]interface{}{
							`lang`:   `en`,
							`active`: true,
						},
					},
				},
				Function: func(in string) (map[string]interface{}, error) {
					if u, err := url.Parse(in); err == nil {
						return maputil.M(u.Query()).MapNative(), nil
					} else {
						return nil, err
					}
				},
			}, {
				Name:    `urlFragment`,
				Summary: `Return the fragment component from the given URL.`,
				Arguments: []funcArg{
					{
						Name:        `url`,
						Type:        `string`,
						Description: `The URL to parse`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `urlFragment "https://example.com:8443/path/to/file.xml?lang=en&active=true#s6.9"`,
						Return: `s6.9`,
					},
				},
				Function: func(in string) (string, error) {
					if u, err := url.Parse(in); err == nil {
						return u.Fragment, nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `hex`,
				Summary: `Encode the given value as a hexadecimal string.`,
				Arguments: []funcArg{
					{
						Name: `input`,
						Type: `string, bytes`,
						Description: `The value to encode. If a byte array is provided, it will be encoded in ` +
							`hexadecimal. If a string is provided, it will converted to a byte array first, then encoded.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `hex "hello"`,
						Return: `68656c6c6f`,
					},
				},
				Function: func(input interface{}) (string, error) {
					return hex.EncodeToString(toBytes(input)), nil
				},
			}, {
				Name:    `base32`,
				Summary: `Encode the given bytes with the Base32 encoding scheme.`,
				Arguments: []funcArg{
					{
						Name: `input`,
						Type: `string, bytes`,
						Description: `The value to encode. If a byte array is provided, it will be encoded directly. ` +
							`If a string is provided, it will converted to a byte array first, then encoded.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `base32 "hello"`,
						Return: `nbswy3dp`,
					},
				},
				Function: func(input interface{}) string {
					return Base32Alphabet.EncodeToString(toBytes(input))
				},
			}, {
				Name:    `base58`,
				Summary: `Encode the given bytes with the Base58 (Bitcoin alphabet) encoding scheme.`,
				Function: func(input interface{}) string {
					return base58.Encode(toBytes(input))
				},
			}, {
				Name:    `base64`,
				Summary: `Encode the given bytes with the Base64 encoding scheme.  Optionally specify the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).`,
				Arguments: []funcArg{
					{
						Name: `input`,
						Type: `string, bytes`,
						Description: `The value to encode. If a byte array is provided, it will be encoded directly. ` +
							`If a string is provided, it will converted to a byte array first, then encoded.`,
					}, {
						Name:        `encoding`,
						Type:        `string`,
						Optional:    true,
						Description: `Specify an encoding option for generating the Base64 representation.`,
						Valid: []funcArg{
							{
								Name:        `standard`,
								Description: `Standard Base-64 encoding scheme, no padding.`,
							}, {
								Name:        `padded`,
								Description: `Standard Base-64 encoding scheme, preserves padding.`,
							}, {
								Name:        `url`,
								Description: `Encoding that can be used in URLs and filenames, no padding.`,
							}, {
								Name:        `url-padded`,
								Description: `Encoding that can be used in URLs and filenames, preserves padding.`,
							},
						},
					},
				},
				Examples: []funcExample{
					{
						Code:   `base64 "hello?yes=this&is=dog#"`,
						Return: `aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw`,
					}, {
						Description: `This is identical to the above example, but with the encoding explicitly specified.`,
						Code:        `base64 "hello?yes=this&is=dog#" "standard"`,
						Return:      `aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw`,
					}, {
						Code:   `base64 "hello?yes=this&is=dog#" "padded"`,
						Return: `aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw==`,
					}, {
						Code:   `base64 "hello?yes=this&is=dog#" "url"`,
						Return: `aGVsbG8_eWVzPXRoaXMmaXM9ZG9nIw`,
					}, {
						Code:   `base64 "hello?yes=this&is=dog#" "url-padded"`,
						Return: `aGVsbG8_eWVzPXRoaXMmaXM9ZG9nIw==`,
					},
				},
				Function: func(input interface{}, encoding ...string) string {
					if len(encoding) == 0 {
						encoding = []string{`standard`}
					}

					switch encoding[0] {
					case `padded`:
						return base64.StdEncoding.EncodeToString(toBytes(input))
					case `url`:
						return base64.RawURLEncoding.EncodeToString(toBytes(input))
					case `url-padded`:
						return base64.URLEncoding.EncodeToString(toBytes(input))
					default:
						return base64.RawStdEncoding.EncodeToString(toBytes(input))
					}
				},
			},

			{
				Name:    `unhex`,
				Summary: `Decode the given hexadecimal string into bytes.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The value to decode into a byte array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `unhex "68656c6c6f"`,
						Return: []byte{'h', 'e', 'l', 'l', 'o'},
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return hex.DecodeString(typeutil.String(input))
				},
			}, {
				Name:    `unbase32`,
				Summary: `Decode the given Base32-encoded string into bytes.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to decode.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `unbase32 "nbswy3dp"`,
						Return: []byte{'h', 'e', 'l', 'l', 'o'},
					},
				},
				Function: func(input interface{}) ([]byte, error) {
					return Base32Alphabet.DecodeString(typeutil.String(input))
				},
			}, {
				Name:    `unbase58`,
				Summary: `Decode the given Base58-encoded string (Bitcoin alphabet) into bytes.`,
				Function: func(input interface{}) []byte {
					return base58.Decode(typeutil.String(input))
				},
			}, {
				Name:    `unbase64`,
				Summary: `Decode the given Base64-encoded string into bytes.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to decode.`,
					}, {
						Name:        `encoding`,
						Type:        `string`,
						Optional:    true,
						Description: `Specify the encoding of the input string.`,
						Valid: []funcArg{
							{
								Name:        `standard`,
								Description: `Standard Base-64 encoding scheme, no padding.`,
							}, {
								Name:        `padded`,
								Description: `Standard Base-64 encoding scheme, preserves padding.`,
							}, {
								Name:        `url`,
								Description: `Encoding that can be used in URLs and filenames, no padding.`,
							}, {
								Name:        `url-padded`,
								Description: `Encoding that can be used in URLs and filenames, preserves padding.`,
							},
						},
					},
				},
				Examples: []funcExample{
					{
						Code:   `unbase64 "aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw"`,
						Return: []byte("hello?yes=this&is=dog#"),
					}, {
						Description: `This is identical to the above example, but with the encoding explicitly specified.`,
						Code:        `unbase64 "aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw" "standard"`,
						Return:      []byte("hello?yes=this&is=dog#"),
					}, {
						Description: `This shows how to convert the binary data to a Unicode string (assuming it is a Unicode bytestream)`,
						Code:        `chr2str (unbase64 "aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw")`,
						Return:      "hello?yes=this&is=dog#",
					}, {
						Code:   `unbase64 "aGVsbG8/eWVzPXRoaXMmaXM9ZG9nIw==" "padded"`,
						Return: []byte("hello?yes=this&is=dog#"),
					}, {
						Code:   `unbase64 "aGVsbG8_eWVzPXRoaXMmaXM9ZG9nIw" "url"`,
						Return: []byte("hello?yes=this&is=dog#"),
					}, {
						Code:   `unbase64 "aGVsbG8_eWVzPXRoaXMmaXM9ZG9nIw==" "url-padded"`,
						Return: []byte("hello?yes=this&is=dog#"),
					},
				},
				Function: func(input interface{}, encoding ...string) ([]byte, error) {
					s := typeutil.String(input)

					if len(encoding) == 0 {
						if strings.Contains(s, `=`) {
							encoding = []string{`padded`}
						} else {
							encoding = []string{`standard`}
						}
					}

					switch encoding[0] {
					case `padded`:
						return base64.StdEncoding.DecodeString(s)
					case `url`:
						return base64.RawURLEncoding.DecodeString(s)
					case `url-padded`:
						return base64.URLEncoding.DecodeString(s)
					default:
						return base64.RawStdEncoding.DecodeString(s)
					}
				},
			}, {
				Name:    `httpStatusText`,
				Summary: `Return a human-readable description of the given HTTP error code.`,
				Function: func(code interface{}) string {
					return http.StatusText(int(typeutil.Int(code)))
				},
				Examples: []funcExample{
					{
						Code:   `httpStatusText 404`,
						Return: `Not Found`,
					}, {
						Code:   `httpStatusText "404"`,
						Return: `Not Found`,
					}, {
						Code:   `httpStatusText 979`,
						Return: ``,
					},
				},
			}, {
				Name:    `chr2str`,
				Summary: `Takes an array of integers representing Unicode codepoints and returns the resulting string.`,
				Function: func(codepoints interface{}) string {
					var points = sliceutil.Sliceify(codepoints)
					var chars = make([]rune, len(points))

					for i, n := range points {
						if codepoint := int(typeutil.Int(n)); codepoint > 0 {
							chars[i] = rune(codepoint)
						}
					}

					return string(chars)
				},
				Examples: []funcExample{
					{
						Code:   `chr2str [72, 69, 76, 76, 79]`,
						Return: `HELLO`,
					}, {
						Code:   `chr2str [84, 72, 69, 82, 69]`,
						Return: `THERE`,
					},
				},
			},
		},
	}
}
