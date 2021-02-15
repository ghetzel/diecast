package diecast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLayerShouldConsiderOpening(t *testing.T) {
	var layer = Layer{}

	assert.True(t, layer.shouldConsiderOpening(`/hello`))

	layer.Paths = []string{
		`/other`,
		`/other/**`,
	}

	assert.True(t, layer.shouldConsiderOpening(`/other`))
	assert.True(t, layer.shouldConsiderOpening(`/other/`))
	assert.True(t, layer.shouldConsiderOpening(`/other/deeply/nested/file.txt`))
	assert.False(t, layer.shouldConsiderOpening(`/hello`))
}
