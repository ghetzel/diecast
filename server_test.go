package diecast

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/testify/require"
)

func doTestServerRequest(s *Server, method string, path string, tester func(*httptest.ResponseRecorder)) {
	var req = httptest.NewRequest(method,
		fmt.Sprintf("http://%s%s", DefaultAddress, path), nil)

	req.Header.Set(`X-Diecast-Binding`, `test`)

	var w = httptest.NewRecorder()
	s.ServeHTTP(w, req)

	// if w.Code >= 400 {
	// 	log.Errorf("Response %d: %s", w.Code, w.Body.String())
	// }

	tester(w)
}

func TestStaticServer(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/hello`)
	var mounts = getTestMounts(assert)

	server.SetMounts(mounts)
	assert.Nil(server.Initialize())
	assert.Equal(len(mounts), len(server.Mounts))
	assert.Equal(`http://127.0.0.1:28419`, server.LocalURL())

	doTestServerRequest(server, `GET`, `/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(w.Body.String(), `Hello`)
		})

	doTestServerRequest(server, `GET`, `/css/bootstrap.min.css`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Contains(string(data[:]), `Bootstrap`)
		})

	doTestServerRequest(server, `GET`, `/js/jquery.min.js`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Contains(string(data[:]), `jQuery`)
		})
}

func TestStaticServerWithRoutePrefix(t *testing.T) {
	log.SetLevelString(`debug`)
	var assert = require.New(t)
	var server = NewServer(`./tests/hello`)
	server.RoutePrefix = `/ui`
	var mounts = getTestMounts(assert)
	server.SetMounts(mounts)
	assert.Nil(server.Initialize())
	assert.Equal(len(mounts), len(server.Mounts))

	doTestServerRequest(server, `GET`, `/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/css/bootstrap.min.css`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/ui/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(w.Body.String(), `Hello`)
		})

	doTestServerRequest(server, `GET`, `/ui/js/jquery.min.js`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Contains(string(data[:]), `jQuery`)
		})

	doTestServerRequest(server, `GET`, `/ui/css/bootstrap.min.css`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Contains(string(data[:]), `Bootstrap`)
		})
}

func TestStaticServerTemplateSomethingInMount(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/hello`, `*.txt`)
	server.SetMounts(getTestMounts(assert))

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("GET\n", string(data[:]))
		})

	doTestServerRequest(server, `POST`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("POST\n", string(data[:]))
		})
}

func TestStaticServerTemplateSomethingInMountWithRoutePrefix(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/hello`, `*.txt`)
	server.RoutePrefix = `/ui`
	server.SetMounts(getTestMounts(assert))

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `POST`, `/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/ui/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("GET\n", string(data[:]))
		})

	doTestServerRequest(server, `POST`, `/ui/test/should-render.txt`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("POST\n", string(data[:]))
		})
}

func TestFilesInRootSubdirectories(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/test_root1`, `*.html`)
	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/subdir1/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(w.Body.String(), `Hello`)
		})

	doTestServerRequest(server, `GET`, `/subdir1/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			assert.Contains(w.Body.String(), `Hello`)
		})
}

func TestFilesInMountSubdirectories(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/hello`, `*.html`, `*.txt`)
	server.SetMounts(getTestMounts(assert))

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/test/subdir1`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(301, w.Code)
		})

	doTestServerRequest(server, `GET`, `/test/subdir1/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(404, w.Code)
		})

	doTestServerRequest(server, `GET`, `/test/subdir1/test.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("<h1>GET</h1>\n", string(data[:]))
		})

	doTestServerRequest(server, `GET`, `/test/subdir2`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(301, w.Code)
		})

	doTestServerRequest(server, `GET`, `/test/subdir2/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("INDEX GET\n", string(data[:]))
		})

	doTestServerRequest(server, `PUT`, `/test/subdir2/`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("INDEX PUT\n", string(data[:]))
		})

	doTestServerRequest(server, `GET`, `/test/subdir2/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("INDEX GET\n", string(data[:]))
		})

	doTestServerRequest(server, `PUT`, `/test/subdir2/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)
			var data = w.Body.Bytes()
			assert.Equal("INDEX PUT\n", string(data[:]))
		})
}

func TestLayoutsDisabled(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/layouts`, `*.html`)
	server.EnableLayouts = false
	var mounts = getTestMounts(assert)
	server.SetMounts(mounts[3:4])

	assert.Nil(server.Initialize())

	var fn = func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = w.Body.Bytes()
		assert.True(strings.HasPrefix(string(data[:]), "<b>GET</b>"))
	}

	doTestServerRequest(server, `GET`, `/`, fn)
	doTestServerRequest(server, `GET`, `/index.html`, fn)
	doTestServerRequest(server, `GET`, `/layout-test/test1.html`, fn)
}

func TestLayoutsDefault(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/layouts`, `*.html`)
	var mounts = getTestMounts(assert)
	server.SetMounts(mounts[3:4])

	assert.Nil(server.Initialize())

	var fn = func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = strings.TrimSpace(w.Body.String())
		assert.True(strings.HasPrefix(data, "<h1><b>GET</b>"))
	}

	doTestServerRequest(server, `GET`, `/`, fn)
	doTestServerRequest(server, `GET`, `/index.html`, fn)
	doTestServerRequest(server, `GET`, `/layout-test/test1.html`, fn)

	doTestServerRequest(server, `GET`, `/_partial.html`, func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = strings.TrimSpace(w.Body.String())
		assert.Equal("AS-IS", data)
	})

	doTestServerRequest(server, `GET`, `/_partial`, func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = strings.TrimSpace(w.Body.String())
		assert.Equal("AS-IS", data)
	})

	doTestServerRequest(server, `GET`, `/h2layout`, func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = strings.TrimSpace(w.Body.String())
		assert.Equal("<h2><b>GET</b>\n</h2>", data)
	})

	doTestServerRequest(server, `GET`, `/h2-nolayout`, func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = strings.TrimSpace(w.Body.String())
		assert.Equal("<b>GET</b>", data)
	})
}

func TestIncludes(t *testing.T) {
	var assert = require.New(t)
	var server = NewServer(`./tests/layouts`, `*.html`)

	assert.Nil(server.Initialize())

	doTestServerRequest(server, `GET`, `/include-base.html`, func(w *httptest.ResponseRecorder) {
		assert.Equal(200, w.Code)
		var data = strings.TrimSpace(w.Body.String())
		assert.Equal("<b>GET</b>\n<i>GET</i>\n\n<u>GET</u>", data)
	})
}
