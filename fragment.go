package diecast

import (
	"fmt"
	"io"
	"regexp"
)

type Fragment struct {
	Name   string
	Header *TemplateHeader
	Data   []byte
}

type FragmentSet []*Fragment

func (self FragmentSet) Header(server *Server) TemplateHeader {
	var baseHeader TemplateHeader

	if server != nil && server.BaseHeader != nil {
		baseHeader = *server.BaseHeader
		baseHeader.Locale = server.Locale
		baseHeader.Translations = server.Translations
	}

	var finalHeader = &baseHeader

	for _, frag := range self {
		if frag.Header != nil {
			if fh, err := finalHeader.Merge(frag.Header); err == nil {
				finalHeader = fh
			} else {
				panic(err.Error())
			}
		}
	}

	return *finalHeader
}

func (self FragmentSet) FirstLayout() string {
	for _, fragment := range self {
		if fragment.Name != ContentTemplateName {
			return fragment.Name
		}
	}

	return ``
}

func (self FragmentSet) Get(name string) (*Fragment, bool) {
	for _, fragment := range self {
		if fragment.Name == name {
			return fragment, true
		}
	}

	return nil, false
}

func (self *FragmentSet) Set(name string, header *TemplateHeader, data []byte) error {
	if _, ok := self.Get(name); ok {
		return nil
	}

	*self = append(*self, &Fragment{
		Name:   name,
		Header: header,
		Data:   data,
	})

	return nil
}

func (self *FragmentSet) Parse(name string, source io.Reader) error {
	if _, ok := self.Get(name); ok {
		return nil
	}

	if header, data, err := SplitTemplateHeaderContent(source); err == nil {
		*self = append(*self, &Fragment{
			Name:   name,
			Header: header,
			Data:   data,
		})

		return nil
	} else {
		return err
	}
}

func (self *FragmentSet) ChainLayouts() error {
	// loop through the fragments, holding a reference to each one
	// for the next iteration.
	// the next fragment (if there is one) should be what the previous
	// fragment includes IFF it is specifying a "content" include
	//
	// this effectively implements layout single inheritence
	//
	var prevFrag *Fragment

	for _, fragment := range *self {
		if prevFrag == nil {
			prevFrag = fragment
		} else {
			prevFrag.Data = regexp.MustCompile(`\{\{\s+template\s+"`+ContentTemplateName+`"\s+(\S+)\s+\}\}`).ReplaceAll(
				prevFrag.Data,
				[]byte(fmt.Sprintf("{{ template %q $1 }}", fragment.Name)),
			)
		}
	}

	return nil
}

func (self FragmentSet) DebugOutput() []byte {
	var out []byte

	for _, frag := range self {
		out = append(out, []byte(fmt.Sprintf("\n{{/* BEGIN FRAGMENT %q */}}\n", frag.Name))...)
		out = append(out, frag.Data...)
		out = append(out, []byte(fmt.Sprintf("\n{{/* END FRAGMENT %q */}}\n", frag.Name))...)
	}

	return out
}
