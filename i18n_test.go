package diecast

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEyeEighteenEn(t *testing.T) {
	assert := require.New(t)
	server := NewServer(`./tests/i18n`)
	assert.NoError(server.LoadConfig(`./tests/i18n/diecast.yml`))
	assert.NoError(server.Initialize())

	doTestServerRequest(server, `GET`, `/index.html`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			content := string(w.Body.Bytes())
			content = strings.TrimSpace(content)

			assert.Equal(`Hello`, content)
		})

	doTestServerRequest(server, `GET`, `/index.html?lang=es`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			content := string(w.Body.Bytes())
			content = strings.TrimSpace(content)

			assert.Equal(`¡Hola`, content)
		})

	doTestServerRequest(server, `GET`, `/index.html?lang=ru`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			content := string(w.Body.Bytes())
			content = strings.TrimSpace(content)

			assert.Equal(`Привет`, content)
		})

	doTestServerRequest(server, `GET`, `/index.html?lang=zz`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			content := string(w.Body.Bytes())
			content = strings.TrimSpace(content)

			assert.Equal(`Hello`, content)
		})

	server.Locale = `ru`

	doTestServerRequest(server, `GET`, `/index.html?lang=zz`,
		func(w *httptest.ResponseRecorder) {
			assert.Equal(200, w.Code)

			content := string(w.Body.Bytes())
			content = strings.TrimSpace(content)

			assert.Equal(`Привет`, content)
		})
}
