package diecast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextData(t *testing.T) {
	var ctx = NewContext(nil)

	ctx.Set(`hello`, `one`)

	assert.Equal(t, `one`, ctx.Get(`hello`).Value)

	ctx.Push(`hello`, `two`)
	ctx.Push(`hello`, `three`)

	assert.Equal(t, []interface{}{`one`, `two`, `three`}, ctx.Get(`hello`).Value)

	assert.Equal(t, `three`, ctx.Pop(`hello`).String())
	assert.Equal(t, []interface{}{`one`, `two`}, ctx.Get(`hello`).Value)

	assert.Equal(t, `two`, ctx.Pop(`hello`).String())
	assert.Equal(t, []interface{}{`one`}, ctx.Get(`hello`).Value)

	assert.Equal(t, `one`, ctx.Pop(`hello`).String())
	assert.Equal(t, nil, ctx.Get(`hello`).Value)

	assert.Equal(t, ``, ctx.Pop(`hello`).String())
	assert.Equal(t, nil, ctx.Get(`hello`).Value)

	assert.Nil(t, ctx.Pop(`empty`).Value)
	assert.Nil(t, ctx.Pop(`empty`).Value)
	assert.Nil(t, ctx.Pop(`empty`).Value)
}
