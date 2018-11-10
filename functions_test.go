package diecast

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectionFunctions(t *testing.T) {
	assert := require.New(t)
	fns := GetStandardFunctions()

	page := fns[`page`].(func(interface{}, interface{}) int)

	assert.Zero(page(0, 0))
	assert.Zero(page(0, -25))
	assert.Zero(page(0, 25))
	assert.Zero(page(-1, 0))
	assert.Zero(page(0, -1))
	assert.Zero(page(1, 1))
	assert.Equal(0, page(1, 25))
	assert.Equal(25, page(2, 25))
	assert.Equal(50, page(3, 25))
	assert.Equal(75, page(4, 25))
	assert.Equal(100, page(5, 25))
}

func TestMiscFunctions(t *testing.T) {
	assert := require.New(t)
	fns := GetStandardFunctions()

	fn_switch := fns[`switch`].(func(input interface{}, fallback interface{}, pairs ...interface{}) interface{})

	assert.Equal(`1`, fn_switch(`a`, `fallback`, `a`, `1`, `b`, `2`))
	assert.Equal(`2`, fn_switch(`b`, `fallback`, `a`, `1`, `b`, `2`))
	assert.Equal(`fallback`, fn_switch(`c`, `fallback`, `a`, `1`, `b`, `2`))
}
