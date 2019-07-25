package diecast

import (
	"strings"

	"github.com/ghetzel/go-stockutil/stringutil"
)

const DefaultLocale = Locale(`en-us`)

type Locale string

func (self Locale) IsSameCountry(other Locale) bool {
	return self.Country() == other.Country()
}

func (self Locale) Country() Locale {
	c, _ := stringutil.SplitPair(string(self), `-`)
	return Locale(strings.ToLower(c))
}

func (self Locale) Locale() string {
	_, l := stringutil.SplitPair(string(self), `-`)
	return strings.ToLower(l)
}
