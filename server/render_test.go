package server_test

import (
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gobuffalo/packr"
	"github.com/peer-calls/peer-calls/server"
	"github.com/stretchr/testify/assert"
)

func getTemplates(t *testing.T) server.Templates {
	box := packr.NewBox("./templates")
	return server.ParseTemplates(box)
}

func TestRender_redirect(t *testing.T) {
	tpl := server.Templates{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(loggerFactory, tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		http.Redirect(w, r, "/other", 302)
		return "", nil, nil
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/other", w.Header().Get("Location"))
}

func TestRender_success(t *testing.T) {
	tpl := getTemplates(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(loggerFactory, tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "index.html", nil, nil
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRender_notFound(t *testing.T) {
	tpl := getTemplates(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(loggerFactory, tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "nonexisting.html", nil, nil
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRender_error(t *testing.T) {
	tpl := getTemplates(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(loggerFactory, tpl, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "index.html", nil, fmt.Errorf("test error")
	}).ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRender_templateError(t *testing.T) {
	templates := server.Templates{}
	tpl := template.New("test.html")
	templates["test.html"] = template.Must(tpl.Parse("<h1>{{.Data.A.B}}</h1>"))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	renderer := server.NewRenderer(loggerFactory, templates, "/test", "v0.0.0")
	renderer.Render(func(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
		return "test.html", struct{ A *string }{A: nil}, nil
	}).ServeHTTP(w, r)
	t.Log(w.Body)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
