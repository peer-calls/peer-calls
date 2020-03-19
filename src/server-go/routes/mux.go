package routes

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
	"github.com/jeremija/peer-calls/src/server-go/render"
)

func joinURL(base string, paths ...string) string {
	p := path.Join(paths...)
	return fmt.Sprintf("%s/%s", strings.TrimRight(base, "/"), strings.TrimLeft(p, "/"))
}

type Mux struct {
	BaseURL    string
	mux        *http.ServeMux
	iceServers string
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.mux.ServeHTTP(w, r)
}

func NewMux(baseURL string, version string, iceServersJSON string) *Mux {
	serveMux := http.NewServeMux()

	box := packr.NewBox("../templates")
	templates := render.ParseTemplates(box)
	renderer := render.NewRenderer(templates, baseURL, version)

	mux := &Mux{
		BaseURL:    baseURL,
		mux:        serveMux,
		iceServers: iceServersJSON,
	}

	serveMux.Handle(joinURL(baseURL, "/"), renderer.Render(mux.routeIndex))
	serveMux.Handle(joinURL(baseURL, "/static"), static("../../../build"))
	serveMux.Handle(joinURL(baseURL, "/call"), http.HandlerFunc(mux.routeNewCall))
	serveMux.Handle(joinURL(baseURL, "/call/*"), renderer.Render(mux.routeCall))

	return mux
}

func static(path string) http.Handler {
	box := packr.NewBox(path)
	return http.FileServer(http.FileSystem(box))
}

func (mux *Mux) routeNewCall(w http.ResponseWriter, r *http.Request) {
	callID := r.PostFormValue("call")
	if callID == "" {
		callID = uuid.New().String()
	}
	url := mux.BaseURL + "/call/" + url.PathEscape(callID)
	http.Redirect(w, r, url, 302)
}

func (mux *Mux) routeIndex(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
	return "index.html", nil, nil
}

func (mux *Mux) routeCall(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
	callID := url.PathEscape(path.Base(r.URL.Path))
	userID := uuid.New().String()
	data := map[string]string{
		"CallID":     callID,
		"UserID":     userID,
		"ICEServers": mux.iceServers,
	}
	return "call.html", data, nil
}
