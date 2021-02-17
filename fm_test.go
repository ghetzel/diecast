package diecast

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/stretchr/testify/assert"
)

func rcstr(s string) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBufferString(s))
}

func TestSplitFrontMatter(t *testing.T) {
	// -------------------------------------------------------------------------------------------------------------------
	var rc, fm, err = SplitFrontMatter(nil)

	assert.Equal(t, io.EOF, err)
	assert.Nil(t, rc)
	assert.Nil(t, fm)

	// -------------------------------------------------------------------------------------------------------------------
	rc, fm, err = SplitFrontMatter(rcstr(`<html></html>`))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, typeutil.String(rc))
	assert.Nil(t, fm)

	// -------------------------------------------------------------------------------------------------------------------
	rc, fm, err = SplitFrontMatter(rcstr("---\n---\n<html></html>"))

	assert.NoError(t, err)
	assert.Equal(t, `<html></html>`, typeutil.String(rc))
	assert.Equal(t, &FrontMatter{
		ContentOffset: 4,
	}, fm)
}
