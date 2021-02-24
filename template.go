package diecast

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ghetzel/go-stockutil/typeutil"
	"gopkg.in/yaml.v2"
)

var Delimiters [2]string = [2]string{`{{`, `}}`}
var DefaultEntryPoint = `content`
var DefaultTemplateEngine = `html`

type Template struct {
	Engine        string  `yaml:"engine"`
	EntryPoint    string  `yaml:"entryPoint"`
	DataSources   DataSet `yaml:"dataSources"`
	Layout        string  `yaml:"layout"`
	Filename      string  `yaml:"-"`
	ContentOffset int     `yaml:"-"`
	sha512sum     string
	body          []byte
	buf           *bytes.Buffer
	gotmpl        *goTemplate
	initDone      bool
}

func ParseTemplateString(source string) (*Template, error) {
	return ParseTemplate(bytes.NewBufferString(source))
}

func ParseTemplate(source io.Reader) (*Template, error) {
	if source == nil {
		return nil, io.EOF
	}

	var summer = sha512.New()

	// tee the source to the hasher above for checksumming goodness
	var summedSource = io.TeeReader(source, summer)
	var tmpl = new(Template)
	var fmData []byte

	if data, err := ioutil.ReadAll(summedSource); err == nil {
		var parts = bytes.SplitN(data, FrontMatterSeparator, 3)

		switch len(parts) {
		case 3:
			fmData = parts[1]
			tmpl.body = parts[2]
		case 2:
			fmData = parts[0]
			tmpl.body = parts[1]
		case 1:
			tmpl.body = parts[0]
		}

		tmpl.buf = bytes.NewBuffer(tmpl.body)
		tmpl.ContentOffset = (2 * len(FrontMatterSeparator)) + len(fmData)
		tmpl.sha512sum = hex.EncodeToString(summer.Sum(nil))
	} else {
		return nil, err
	}

	// Only attempt to parse if we actually read any front matter data.
	if len(fmData) > 0 {
		if err := yaml.UnmarshalStrict(fmData, tmpl); err != nil {
			return nil, err
		}
	}

	return tmpl, tmpl.init()
}

// Implement reader interface.
func (self *Template) Read(b []byte) (int, error) {
	if buf := self.buf; buf != nil {
		return buf.Read(b)
	} else {
		return 0, io.EOF
	}
}

// Initialize the template, parsing the data and making the object ready for subsequent calls to Render
func (self *Template) init() error {
	if self.initDone {
		return nil
	}

	var engine = typeutil.OrString(self.Engine, DefaultTemplateEngine)
	// var name = typeutil.OrString(self.Filename, engine+`:`+self.sha512sum)

	if gotmpl, err := parseGoTemplate(self.entryPoint(), engine, self.TemplateString()); err == nil {
		self.gotmpl = gotmpl
		self.initDone = true
		return nil
	} else {
		return err
	}
}

// Return the raw, unrendered template source.
func (self *Template) TemplateString() string {
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
		ctx = NewContext(nil)
	}

	if w == nil {
		w = ctx
	}

	ctx.Debugf("template: entrypoint is %q", self.entryPoint())
	ctx.Debugf("template: attached: %s", strings.Join(self.gotmpl.Names(), `, `))

	return self.gotmpl.ExecuteTemplate(w, self.entryPoint(), ctx.Data())
}

// Returns the SHA512 checksum of the underlying template file.
func (self *Template) Checksum() string {
	return self.sha512sum
}

func (self *Template) AttachFile(name string, r io.Reader) error {
	if tmpl, err := ParseTemplate(r); err == nil {
		// whatever we need to do to merge in the new template header, do it here
		self.DataSources = append(tmpl.DataSources, self.DataSources...)

		// add this new data to our existing template tree and return
		if pt := tmpl.gotmpl.ParseTree(); pt != nil {
			var _, err = self.gotmpl.AddParseTree(name, pt)
			return err
		} else {
			return fmt.Errorf("invalid layout template")
		}
	} else {
		return err
	}
}
