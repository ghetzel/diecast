package diecast

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghetzel/testify/assert"
)

func init() {
	RegisterRendererByGlob(`*.html`, RendererConfig{
		Type: `template`,
		Methods: []string{
			http.MethodGet,
		},
	})
}

func makeRenderTestServer() *Server {
	var server = new(Server)

	server.VFS.AddOverride(`/index.html`, `{{ now "2006-01-02" }}`)
	server.VFS.AddOverride(`/_test.html`, `{{ hello }}`)
	server.VFS.AddOverride(`/_includes/partial.html`, `{{ $ }}`)

	server.VFS.AddOverride(`/include.html`, strings.Join([]string{
		`---`,
		`page:`,
		`  hello: Greetings`,
		`includes:`,
		`- partial.html`,
		`---`,
		`{{ template "partial" "friend" }}`,
	}, "\n"))

	return server
}

func serverGet(t *testing.T, server *Server, method string, path string, want any) {
	var w = httptest.NewRecorder()
	var res, err = server.SimulateRequestWithRecorder(w, method, path, nil, nil, nil)

	assert.NoError(t, err)

	var got = typeutil.String(res.Body)
	assert.Equal(t, want, got)
}

func TestServerRenderPhase(t *testing.T) {
	var server = makeRenderTestServer()

	serverGet(t, server, http.MethodGet, `/`, time.Now().Format("2006-01-02"))
	serverGet(t, server, http.MethodGet, `/_test`, `{{ hello }}`)
	serverGet(t, server, http.MethodGet, `/include`, `friend`)
}
