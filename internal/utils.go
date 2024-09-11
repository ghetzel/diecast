package internal

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
)

var Delimiters [2]string = [2]string{`{{`, `}}`}
var FrontMatterSeparator = []byte("---\n")

type TemplateHeader struct {
	Engine        string                 `yaml:"engine"`
	EntryPoint    string                 `yaml:"entryPoint"`
	DataSources   DataSet                `yaml:"dataSources"`
	Layout        string                 `yaml:"layout"`
	Page          map[string]interface{} `yaml:"page"`
	Filename      string                 `yaml:"-"`
	ContentOffset int                    `yaml:"-"`
	SHA512SUM     string                 `yaml:"-"`
}

func SplitTemplateHeaderContent(r io.Reader) (*TemplateHeader, []byte, error) {
	var summer = sha512.New()
	var hdr = new(TemplateHeader)
	var body []byte
	var fmData []byte

	// tee the source to the hasher above for checksumming goodness
	var summedSource = io.TeeReader(r, summer)

	if data, err := ioutil.ReadAll(summedSource); err == nil {
		var parts = bytes.SplitN(data, FrontMatterSeparator, 3)

		switch len(parts) {
		case 3:
			fmData = parts[1]
			body = parts[2]
		case 2:
			fmData = parts[0]
			body = parts[1]
		case 1:
			body = parts[0]
		}

		hdr.ContentOffset = (2 * len(FrontMatterSeparator)) + len(fmData)
		hdr.SHA512SUM = hex.EncodeToString(summer.Sum(nil))
	} else {
		return nil, nil, err
	}

	// Only attempt to parse if we actually read any front matter data.
	if len(fmData) > 0 {
		if err := yaml.UnmarshalStrict(fmData, hdr); err != nil {
			return nil, body, err
		}
	}

	return hdr, body, nil
}
