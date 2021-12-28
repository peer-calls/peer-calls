package server_test

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
)

func getTemplates() server.Templates {
	return server.ParseTemplates(embed.Templates)
}

func TestRender_redirect(t *testing.T) {
	t.Parallel()

	tpl := server.Templates{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(test.NewLogger(), tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		http.Redirect(w, r, "/other", http.StatusFound)
		return "", nil, nil
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/other", w.Header().Get("Location"))
}

func TestRender_success(t *testing.T) {
	t.Parallel()

	tpl := getTemplates()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(test.NewLogger(), tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "index.html", nil, nil
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRender_notFound(t *testing.T) {
	t.Parallel()

	tpl := getTemplates()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(test.NewLogger(), tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "nonexisting.html", nil, nil
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRender_error(t *testing.T) {
	t.Parallel()

	tpl := getTemplates()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(test.NewLogger(), tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "index.html", nil, errors.Errorf("test error")
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRender_templateError(t *testing.T) {
	t.Parallel()

	templates := server.Templates{}
	tpl := template.New("test.html")
	templates["test.html"] = template.Must(tpl.Parse("<h1>{{.Data.A.B}}</h1>"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(test.NewLogger(), templates, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "test.html", struct{ A *string }{A: nil}, nil
	}).ServeHTTP(w, r)
	t.Log(w.Body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
