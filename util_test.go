package diecast

import (
	"testing"

	"github.com/ghetzel/testify/require"
)

func TestFancyMapJoin(t *testing.T) {
	var assert = require.New(t)

	assert.Equal(`hello=there`, fancyMapJoin(map[string]interface{}{
		`hello`: `there`,
	}))

	assert.Equal(`hello=there&how=are you?`, fancyMapJoin(map[string]interface{}{
		`hello`: `there`,
		`how`:   `are you?`,
	}))

	assert.Equal(`hello: there; how: are you?`, fancyMapJoin(map[string]interface{}{
		`_kvjoin`: `: `,
		`_join`:   `; `,
		`hello`:   `there`,
		`how`:     `are you?`,
	}))

	assert.Equal(`hello: "there"; how: "are you?"`, fancyMapJoin(map[string]interface{}{
		`_kvjoin`:  `: `,
		`_join`:    `; `,
		`_vformat`: "%q",
		`hello`:    `there`,
		`how`:      `are you?`,
	}))
}
