package diecast

import (
	"bytes"
	"io"
	"testing"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/stretchr/testify/assert"
)

func sr(s string) io.Reader {
	return bytes.NewBufferString(s)
}

func TestParseTemplate(t *testing.T) {
	// -------------------------------------------------------------------------------------------------------------------
	var tmpl, unread, err = ParseTemplate(nil)

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, tmpl)
	assert.Nil(t, unread)

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, unread, err = ParseTemplate(sr(`<html></html>`))

	assert.NoError(t, err)
	assert.Nil(t, tmpl)
	assert.Equal(t, `<html></html>`, fileutil.Cat(unread))

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, unread, err = ParseTemplate(sr("---\n---\n<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, tmpl.templateString())
	assert.Equal(t, `<html></html>`, tmpl.String())
	assert.Equal(t, 8, tmpl.ContentOffset)
	assert.Equal(t, `c86b225eb1e395b4e33a21fd17ae78adecd0bdce7cd0297d8af559b23ef5b9e3a4e6d0e050e815311e45394297fe87373bb250a93f2039d98698f2603e99a262`, tmpl.Checksum())
}
