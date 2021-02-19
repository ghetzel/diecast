package diecast

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"io/ioutil"

	"github.com/ghetzel/go-stockutil/typeutil"
	"gopkg.in/yaml.v2"
)

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
}

func ParseTemplate(source io.Reader) (*Template, io.Reader, error) {
	if source == nil {
		return nil, nil, io.EOF
	}

	var chunk = make([]byte, len(FrontMatterSeparator))

	// Front Matter is declared in the first 4 bytes of the file being `---\n`.  If this is not the case,
	// then we kinda glue that first 4 bytes back in place and return an intact, (effectively) unread io.Reader.
	if n, err := io.ReadFull(source, chunk); err == nil {
		if bytes.Equal(chunk, FrontMatterSeparator) {
			var summer = sha512.New()
			var tmpl Template
			var fmData []byte

			// tee the source to the hasher above for checksumming goodness
			source = io.TeeReader(source, summer)

			if data, err := ioutil.ReadAll(source); err == nil {
				var parts = bytes.SplitN(data, FrontMatterSeparator, 2)

				if len(parts) == 2 {
					fmData = parts[0]
					tmpl.body = parts[1]
				} else {
					tmpl.body = parts[0]
				}

				tmpl.ContentOffset = len(FrontMatterSeparator) + len(fmData)
				tmpl.sha512sum = hex.EncodeToString(summer.Sum(nil))
			} else {
				return nil, nil, err
			}

			// Only attempt to parse if we actually read any front matter data.
			if len(fmData) > 0 {
				if err := yaml.UnmarshalStrict(fmData, &tmpl); err != nil {
					return nil, nil, err
				}
			}

			return &tmpl, nil, tmpl.init()
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

func (self *Template) init() error {
	var engine = typeutil.OrString(self.Engine, DefaultTemplateEngine)
	var name = typeutil.OrString(self.Filename, engine+`:`+self.sha512sum)

	if gotmpl, err := parseGoTemplate(name, engine, self.templateString()); err == nil {
		self.gotmpl = gotmpl
		return nil
	} else {
		return err
	}
}

func (self *Template) templateString() string {
	return string(self.body)
}

func (self *Template) String() string {
	return string(self.body)
}

func (self *Template) Render(w io.Writer) error {
	var entryPoint = typeutil.OrString(self.EntryPoint, DefaultEntryPoint)

	...
}
