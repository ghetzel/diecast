package diecast

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func rcstr(s string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString(s))
}

func TestParseTemplate(t *testing.T) {
	// -------------------------------------------------------------------------------------------------------------------
	var tmpl, err = ParseTemplate(nil)

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, tmpl)

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(rcstr(`<html></html>`))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, tmpl.String())
	assert.Nil(t, tmpl)

	// -------------------------------------------------------------------------------------------------------------------
	tmpl, err = ParseTemplate(rcstr("---\n---\n<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, tmpl.String())
	assert.Equal(t, 4, tmpl.ContentOffset)
}
