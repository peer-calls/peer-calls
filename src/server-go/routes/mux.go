package routes

import (
	"html/template"
	"net/http"
	"net/url"
	"path"

	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr"
	"github.com/google/uuid"
	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/render"
	"github.com/jeremija/peer-calls/src/server-go/routes/wsserver"
)

type Mux struct {
	BaseURL    string
	handler    *chi.Mux
	iceServers string
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.handler.ServeHTTP(w, r)
}

func NewMux(
	baseURL string,
	version string,
	iceServers []config.ICEServer,
	iceServersJSON string,
	rooms RoomManager,
	tracks TracksManager,
) *Mux {
	box := packr.NewBox("../templates")
	templates := render.ParseTemplates(box)
	renderer := render.NewRenderer(templates, baseURL, version)

	handler := chi.NewRouter()
	mux := &Mux{
		BaseURL:    baseURL,
		handler:    handler,
		iceServers: iceServersJSON,
	}

	var root string
	if baseURL == "" {
		root = "/"
	} else {
		root = baseURL
	}

	handler.Route(root, func(router chi.Router) {
		router.Get("/", renderer.Render(mux.routeIndex))
		router.Handle("/static/*", static(baseURL+"/static", "../../../build"))
		router.Handle("/res/*", static(baseURL+"/res", "../../../res"))
		router.Post("/call", mux.routeNewCall)
		router.Get("/call/{callID}", renderer.Render(mux.routeCall))

		router.Mount("/ws", NewPeerToPeerRoomHandler(wsserver.NewWSS(rooms)))
		router.Mount("/ws-server", NewPeerToServerRoomHandler(wsserver.NewWSS(rooms), iceServers, tracks))
	})

	return mux
}

func static(prefix string, path string) http.Handler {
	box := packr.NewBox(path)
	fileServer := http.FileServer(http.FileSystem(box))
	return http.StripPrefix(prefix, fileServer)
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
	data := map[string]interface{}{
		"CallID":     callID,
		"UserID":     userID,
		"ICEServers": template.HTML(mux.iceServers),
	}
	return "call.html", data, nil
}
