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
	var tmpl, err = ParseTemplate(nil)

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, tmpl)

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(sr(`<html></html>`))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, fileutil.Cat(tmpl))

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(sr("---\n---\n<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, tmpl.templateString())
	assert.Equal(t, `<html></html>`, tmpl.String())
	assert.Equal(t, 8, tmpl.ContentOffset)
	assert.Equal(t, `c86b225eb1e395b4e33a21fd17ae78adecd0bdce7cd0297d8af559b23ef5b9e3a4e6d0e050e815311e45394297fe87373bb250a93f2039d98698f2603e99a262`, tmpl.Checksum())

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(sr("---\n<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, tmpl.templateString())
	assert.Equal(t, `<html></html>`, tmpl.String())
	assert.Equal(t, 8, tmpl.ContentOffset)
	assert.Equal(t, `fc15f673c12dafb0a3f4600429742a85eebb4be5447cd0ca23a6cb37aea68541b8310bac1ed331ed984d57fcc8a77823c6151be6a37f272404c991295551f4ea`, tmpl.Checksum())

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(sr("<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, tmpl.templateString())
	assert.Equal(t, `<html></html>`, tmpl.String())
	assert.Equal(t, 8, tmpl.ContentOffset)
	assert.Equal(t, `83bafe4c888008afdd1b72c028c7f50dee651ca9e7d8e1b332e0bf3aa1315884155a1458a304f6e5c5627e714bf5a855a8b8d7db3f4eb2bb2789fe2f8f6a1d83`, tmpl.Checksum())

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(sr("entryPoint: hello\n---\n<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `hello`, tmpl.EntryPoint)
}
