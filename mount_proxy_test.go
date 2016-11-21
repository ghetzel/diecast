package diecast

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestMountProxyPlain(t *testing.T) {
	assert := require.New(t)
	mounts := getTestMounts(assert)
	proxy := MountProxy{
		Fallback: http.Dir(`./examples/hello`),
		Mounts:   mounts,
	}

	assert.Equal(proxy.FindMountForEndpoint(`/js/bootstrap.min.js`).MountPoint, mounts[0].MountPoint)
	assert.Equal(proxy.FindMountForEndpoint(`/css/bootstrap.min.css`).MountPoint, mounts[1].MountPoint)
	assert.Nil(proxy.FindMountForEndpoint(`/nonexistent`))
	assert.Nil(proxy.FindMountForEndpoint(`/index.html`))
}
