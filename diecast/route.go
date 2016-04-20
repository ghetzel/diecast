package diecast

import (
	"github.com/shutterstock/go-stockutil/sliceutil"
	"net/http"
	"regexp"
	"strings"
)

type Route struct {
	Path    string            `json:"path"`
	Methods []string          `json:"methods,omitempty"`
	Headers map[string]string `json:"headers"`
	Final   bool              `json:"final,omitempty"`
}

func (self *Route) IsMatch(method string, routePath string, req *http.Request) bool {
	if sliceutil.ContainsString(self.Methods, strings.ToLower(method)) {
		if ok, err := regexp.MatchString(self.Path, routePath); err == nil && ok {
			return true
		}
	}

	return false
}

func (self *Route) Apply(w http.ResponseWriter) error {
	for name, value := range self.Headers {
		w.Header().Set(name, value)
	}

	return nil
}
