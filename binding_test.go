package diecast

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghetzel/testify/require"
)

func TestBindingHttp(t *testing.T) {
	assert := require.New(t)
	mux := http.NewServeMux()
	dc := NewServer(`./tests/hello`)
	funcs := dc.GetTemplateFunctions(make(map[string]interface{}), nil)

	mux.HandleFunc(`/test/thing.json`, func(w http.ResponseWriter, req *http.Request) {
		httputil.RespondJSON(w, map[string]interface{}{
			`success`: `ok`,
		})
	})

	mux.HandleFunc(`/test/code/`, func(w http.ResponseWriter, req *http.Request) {
		code := typeutil.Int(strings.TrimPrefix(req.URL.Path, `/test/code/`))

		httputil.RespondJSON(w, map[string]interface{}{
			`code`: code,
		}, int(code))

	})

	server := httptest.NewServer(mux)

	// status code tests
	// ---------------------------------------------------------------------------------------------
	log.Noticef("%v/test/code/200", server.URL)

	binding := &Binding{
		Name:     `test1`,
		Resource: fmt.Sprintf("%v/test/code/200", server.URL),
		server:   dc,
	}

	out, err := binding.Evaluate(
		httptest.NewRequest(`GET`, `/test/code/200`, nil),
		&TemplateHeader{},
		make(map[string]interface{}),
		funcs,
	)

	assert.NoError(err)
	assert.Equal(map[string]interface{}{
		`code`: float64(200),
	}, out)
}

func TestBindingRedis(t *testing.T) {
	assert := require.New(t)
	redis, err := miniredis.Run()
	assert.NoError(err)
	assert.NotNil(redis)
	defer redis.Close()

	dc := NewServer(`./tests/hello`)
	funcs := dc.GetTemplateFunctions(make(map[string]interface{}), nil)

	redis.Set(`key.1`, `foo`)
	redis.Set(`key.2`, `bar`)
	redis.Set(`key.3`, `baz`)

	redis.HSet(`obj`, `key1`, `foof`)

	binding := &Binding{
		Name:   `testR1`,
		server: dc,
	}

	for i, v := range []string{`foo`, `bar`, `baz`} {
		binding.Resource = fmt.Sprintf("redis://%v/key.%d", redis.Addr(), i+1)
		out, err := binding.Evaluate(
			httptest.NewRequest(`GET`, `/yay`, nil),
			&TemplateHeader{},
			make(map[string]interface{}),
			funcs,
		)

		assert.NoError(err)
		assert.Equal(v, out)
	}

	binding = &Binding{
		Name:     `testR1`,
		Method:   `HGETALL`,
		Resource: fmt.Sprintf("redis://%v/obj", redis.Addr()),
		server:   dc,
	}

	out, err := binding.Evaluate(
		httptest.NewRequest(`GET`, `/yay`, nil),
		&TemplateHeader{},
		make(map[string]interface{}),
		funcs,
	)

	assert.NoError(err)
	assert.Equal(map[string]interface{}{
		`key1`: `foof`,
	}, out)
}
