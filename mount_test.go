package diecast

import (
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
)

func getTestMounts(tt *require.Assertions) []Mount {
	mounts := []Mount{
		{
			Path:       `./examples/external_path/js`,
			MountPoint: `/js`,
		}, {
			Path:       `./examples/external_path/css`,
			MountPoint: `/css`,
		}, {
			Path:       `./examples/external_path/testfiles`,
			MountPoint: `/test`,
		}, {
			Path:       `./examples/external_path/mounted-layouts`,
			MountPoint: `/layout-test`,
		},
	}

	for _, mount := range mounts {
		if err := mount.Initialize(); err != nil {
			tt.NotNil(err)
		}
	}

	return mounts
}

func TestMounts(t *testing.T) {
	assert := require.New(t)
	mounts := getTestMounts(assert)

	var mount Mount
	var file http.File
	var err error
	var data []byte

	// MOUNT 0
	// --------------------------------------------------------------------------------------------
	mount = mounts[0]

	assert.True(mount.WillRespondTo(`/js/bootstrap.min.js`))
	assert.True(mount.WillRespondTo(`/js/jquery.min.js`))
	assert.True(mount.WillRespondTo(`/js/nonexistent.whatever`))
	assert.False(mount.WillRespondTo(`/css/bootstrap.min.css`))
	assert.False(mount.WillRespondTo(`/index.html`))
	assert.False(mount.WillRespondTo(`/`))

	// file read test
	file, err = mount.Open(`/js/bootstrap.min.js`)
	assert.Nil(err)

	data, err = ioutil.ReadAll(file)
	assert.Nil(err)
	assert.NotEmpty(data)
	assert.Contains(string(data[:]), `Bootstrap`)

	// nonexistent file error test
	file, err = mount.Open(`/js/nonexistent.whatever`)
	assert.NotNil(err)

	// MOUNT 1
	// --------------------------------------------------------------------------------------------
	mount = mounts[1]

	assert.True(mount.WillRespondTo(`/css/bootstrap.min.css`))
	assert.False(mount.WillRespondTo(`/js/bootstrap.min.js`))
	assert.False(mount.WillRespondTo(`/index.html`))
	assert.False(mount.WillRespondTo(`/`))
}
