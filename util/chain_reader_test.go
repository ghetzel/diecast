package util

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

var testTextLong = `Duis vitae feugiat sem, a tempor dui. Sed ac tellus nisi. Donec condimentum leo dolor, quis volutpat lectus euismod id. Sed vehicula dolor eu blandit vulputate. Aliquam ullamcorper vitae eros eleifend ultricies. Vestibulum fringilla blandit vestibulum. Curabitur vel commodo massa. Sed id tristique dui, eget tempor nulla. Vestibulum lectus orci, cursus id libero efficitur`

func TestChainableReader(t *testing.T) {
	assert := require.New(t)
	input := bytes.NewBufferString(testTextLong)

	var buf bytes.Buffer

	n, err := io.CopyN(&buf, input, 16)
	assert.NoError(err)
	assert.Equal(16, int(n))
	assert.Equal(testTextLong[0:16], string(buf.Bytes()))

	chain := NewChainableReader(&buf, input)

	output, err := ioutil.ReadAll(chain)
	assert.NoError(err)
	assert.Equal(testTextLong, string(output))
}
