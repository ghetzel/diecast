package diecast

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

type TestFileSystem map[string]http.File

func (self TestFileSystem) Open(name string) (http.File, error) {
	fmt.Printf("Opening %q\n", name)

	if file, ok := self[name]; ok {
		return file, nil
	}

	return nil, os.ErrNotExist
}

func getTestMounts(tt *require.Assertions) []Mount {
	mounts := []Mount{
		&FileMount{
			Path:       `./examples/external_path/js`,
			MountPoint: `/js`,
		},
		&FileMount{
			Path:       `./examples/external_path/css`,
			MountPoint: `/css`,
		},
		&FileMount{
			Path:       `./examples/external_path/testfiles`,
			MountPoint: `/test`,
		},
		&FileMount{
			Path:       `./examples/external_path/mounted-layouts`,
			MountPoint: `/layout-test`,
		},
		&FileMount{
			MountPoint: `/fs-test`,
			FileSystem: TestFileSystem{
				`/first`:  nil,
				`/second`: nil,
				`/third`:  nil,
			},
		},
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

	// MOUNT 4: Custom FileSystem test
	// --------------------------------------------------------------------------------------------
	mount = mounts[4]

	assert.True(mount.WillRespondTo(`/fs-test/first`))
	assert.True(mount.WillRespondTo(`/fs-test/second`))
	assert.True(mount.WillRespondTo(`/fs-test/third`))

	_, err = mount.Open(`/fs-test/first`)
	assert.Nil(err)

	_, err = mount.Open(`/fs-test/second`)
	assert.Nil(err)

	_, err = mount.Open(`/fs-test/third`)
	assert.Nil(err)

	_, err = mount.Open(`/fs-test/NOPE`)
	assert.Equal(os.ErrNotExist, err)
}
