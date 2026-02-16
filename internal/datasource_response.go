package internal

import (
	"encoding/json"
	"io"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"go.yaml.in/yaml/v2"
)

type ResponseParserFunc func(data io.Reader) (any, error)

var responseParsers = make(map[string]ResponseParserFunc)

func init() {
	RegisterResponseParser(`binary`, func(data io.Reader) (any, error) {
		return fileutil.ReadAll(data)
	})

	RegisterResponseParser(`string`, func(data io.Reader) (any, error) {
		return typeutil.String(data), nil
	})

	RegisterResponseParser(`html`, func(data io.Reader) (any, error) {
		return goquery.NewDocumentFromReader(data)
	})

	RegisterResponseParser(`json`, func(data io.Reader) (any, error) {
		var out any

		if err := json.NewDecoder(data).Decode(&out); err == nil {
			return &out, nil
		} else {
			return nil, err
		}
	})

	RegisterResponseParser(`yaml`, func(data io.Reader) (any, error) {
		var out any

		if err := yaml.NewDecoder(data).Decode(&out); err == nil {
			return &out, nil
		} else {
			return nil, err
		}
	})
}

func RegisterResponseParser(key string, fn ResponseParserFunc) {
	responseParsers[key] = fn
}
