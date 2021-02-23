package diecast

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ghetzel/go-stockutil/typeutil"
	"gopkg.in/yaml.v2"
)

var Delimiters [2]string = [2]string{`{{`, `}}`}
var DefaultEntryPoint = `main`
var DefaultTemplateEngine = `html`

type Template struct {
	Engine        string  `yaml:"engine"`
	EntryPoint    string  `yaml:"entryPoint"`
	DataSources   DataSet `yaml:"dataSources"`
	Filename      string  `yaml:"-"`
	ContentOffset int     `yaml:"-"`
	sha512sum     string
	body          []byte
	gotmpl        *goTemplate
	initDone      bool
}

func ParseTemplateString(source string) (*Template, io.Reader, error) {
	return ParseTemplate(bytes.NewBufferString(source))
}

func ParseTemplate(source io.Reader) (*Template, io.Reader, error) {
	if source == nil {
		return nil, nil, io.EOF
	}

	var chunk = make([]byte, len(FrontMatterSeparator))
	var summer = sha512.New()

	// tee the source to the hasher above for checksumming goodness
	var summedSource = io.TeeReader(source, summer)

	// Front Matter is declared in the first 4 bytes of the file being `---\n`.  If this is not the case,
	// then we kinda glue that first 4 bytes back in place and return an intact, (effectively) unread io.Reader.
	if n, err := io.ReadFull(summedSource, chunk); err == nil {
		if bytes.Equal(chunk, FrontMatterSeparator) {
			var tmpl = new(Template)
			var fmData []byte

			if data, err := ioutil.ReadAll(summedSource); err == nil {
				var parts = bytes.SplitN(data, FrontMatterSeparator, 2)

				if len(parts) == 2 {
					fmData = parts[0]
					tmpl.body = parts[1]
				} else {
					tmpl.body = parts[0]
				}

				tmpl.ContentOffset = (2 * len(FrontMatterSeparator)) + len(fmData)
				tmpl.sha512sum = hex.EncodeToString(summer.Sum(nil))
			} else {
				return nil, nil, err
			}

			// Only attempt to parse if we actually read any front matter data.
			if len(fmData) > 0 {
				if err := yaml.UnmarshalStrict(fmData, tmpl); err != nil {
					return nil, nil, err
				}
			}

			return tmpl, nil, tmpl.init()
		} else {
			// paste the bit we just read pack onto the front of the io.Reader like nothing happened
			return nil, io.MultiReader(
				bytes.NewBuffer(chunk[0:n]),
				source,
			), nil
		}
	} else {
		return nil, nil, err
	}
}

// Initialize the template, parsing the data and making the object ready for subsequent calls to Render
func (self *Template) init() error {
	if self.initDone {
		return nil
	}

	var engine = typeutil.OrString(self.Engine, DefaultTemplateEngine)
	// var name = typeutil.OrString(self.Filename, engine+`:`+self.sha512sum)

	if gotmpl, err := parseGoTemplate(self.entryPoint(), engine, self.templateString()); err == nil {
		self.gotmpl = gotmpl
		self.initDone = true
		return nil
	} else {
		return err
	}
}

// Internal stringifyer convenience function.
func (self *Template) templateString() string {
	return string(self.body)
}

// Implement fmt.Stringer
func (self *Template) String() string {
	if err := self.init(); err == nil {
		var dst bytes.Buffer

		if err := self.Render(nil, &dst); err == nil {
			return dst.String()
		} else {
			return fmt.Sprintf("<!-- TEMPLATE ERROR: %v -->", err)
		}
	} else {
		return ``
	}
}

func (self *Template) entryPoint() string {
	return typeutil.OrString(self.EntryPoint, DefaultEntryPoint)
}

// Refresh all data sources and render the template, writing the results to the giveni io.Writer.
func (self *Template) Render(ctx *Context, w io.Writer) error {
	if err := self.init(); err != nil {
		return err
	}

	if ctx == nil {
		ctx = NewContext(nil, nil, nil)
	}

	if w == nil {
		w = ctx
	}

	return self.gotmpl.ExecuteTemplate(w, self.entryPoint(), ctx.MapNative())
}

// Returns the SHA512 checksum of the underlying template file.
func (self *Template) Checksum() string {
	return self.sha512sum
}
