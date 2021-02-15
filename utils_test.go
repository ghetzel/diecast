package diecast

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockHttpFile(t *testing.T) {
	var file *mockHttpFile
	var err error

	file, err = newMockHttpFile(`test.txt`, nil)
	assert.Equal(t, ErrNotFound, err)

	file, err = newMockHttpFile(`test.txt`, `HELLO`)
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())

	file, err = newMockHttpFile(`test.txt`, []byte(`HELLO`))
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())

	file, err = newMockHttpFile(`test.txt`, bytes.NewBufferString(`HELLO`))
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())

	file, err = newMockHttpFile(`test.txt`, errors.New(`HELLO`))
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())
}
