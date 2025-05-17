package diecast

import (
	"fmt"
	"io"
)

type Fragment struct {
	Name   string
	Header *TemplateHeader
	Data   []byte
}

type FragmentSet []*Fragment

func (fragment FragmentSet) Header(server *Server) TemplateHeader {
	var baseHeader TemplateHeader

	if server != nil && server.BaseHeader != nil {
		baseHeader = *server.BaseHeader
		baseHeader.Locale = server.Locale
		baseHeader.Translations = server.Translations
	}

	var finalHeader = &baseHeader

	for _, frag := range fragment {
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

func (fragment FragmentSet) HasLayout() bool {
	for _, fragment := range fragment {
		if fragment.Name == LayoutTemplateName {
			return true
		}
	}

	return false
}

func (fragment FragmentSet) Get(name string) (*Fragment, bool) {
	for _, fragment := range fragment {
		if fragment.Name == name {
			return fragment, true
		}
	}

	return nil, false
}

func (fragment *FragmentSet) Set(name string, header *TemplateHeader, data []byte) error {
	if _, ok := fragment.Get(name); ok {
		return nil
	}

	*fragment = append(*fragment, &Fragment{
		Name:   name,
		Header: header,
		Data:   data,
	})

	return nil
}

func (fragment *FragmentSet) OverrideData(name string, data []byte) {
	for _, f := range *fragment {
		if f.Name == name {
			f.Data = data
			return
		}
	}
}

func (fragment *FragmentSet) Parse(name string, source io.Reader) error {
	if _, ok := fragment.Get(name); ok {
		return nil
	}

	if header, data, err := SplitTemplateHeaderContent(source); err == nil {
		*fragment = append(*fragment, &Fragment{
			Name:   name,
			Header: header,
			Data:   data,
		})

		return nil
	} else {
		return err
	}
}

func (fragment FragmentSet) DebugOutput() []byte {
	var out []byte

	for _, frag := range fragment {
		out = append(out, []byte(fmt.Sprintf("\n{{/* BEGIN FRAGMENT %q */}}\n", frag.Name))...)
		out = append(out, frag.Data...)
		out = append(out, []byte(fmt.Sprintf("\n{{/* END FRAGMENT %q */}}\n", frag.Name))...)
	}

	return out
}
