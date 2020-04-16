package routes

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"path"

	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr"
	"github.com/jeremija/peer-calls/src/server/basen"
	"github.com/jeremija/peer-calls/src/server/config"
	"github.com/jeremija/peer-calls/src/server/iceauth"
	"github.com/jeremija/peer-calls/src/server/render"
	"github.com/jeremija/peer-calls/src/server/wshandler"
)

type Mux struct {
	BaseURL    string
	handler    *chi.Mux
	iceServers []config.ICEServer
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.handler.ServeHTTP(w, r)
}

func NewMux(
	baseURL string,
	version string,
	networkType config.NetworkType,
	iceServers []config.ICEServer,
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
		iceServers: iceServers,
	}

	var root string
	if baseURL == "" {
		root = "/"
	} else {
		root = baseURL
	}

	wsHandler := newWebSocketHandler(
		networkType,
		wshandler.NewWSS(rooms),
		iceServers,
		tracks,
	)

	handler.Route(root, func(router chi.Router) {
		router.Get("/", renderer.Render(mux.routeIndex))
		router.Handle("/static/*", static(baseURL+"/static", packr.NewBox("../../../build")))
		router.Handle("/res/*", static(baseURL+"/res", packr.NewBox("../../../res")))
		router.Post("/call", mux.routeNewCall)
		router.Get("/call/{callID}", renderer.Render(mux.routeCall))

		router.Mount("/ws", wsHandler)
	})

	return mux
}

func newWebSocketHandler(
	networkType config.NetworkType,
	wss *wshandler.WSS,
	iceServers []config.ICEServer,
	tracks TracksManager,
) http.Handler {
	switch networkType {
	case config.NetworkTypeSFU:
		log.Println("Using network type sfu")
		return NewPeerToServerRoomHandler(wss, iceServers, tracks)
	default:
		log.Println("Using network type mesh")
		return NewPeerToPeerRoomHandler(wss)
	}
}

func static(prefix string, box packr.Box) http.Handler {
	fileServer := http.FileServer(http.FileSystem(box))
	return http.StripPrefix(prefix, fileServer)
}

func (mux *Mux) routeNewCall(w http.ResponseWriter, r *http.Request) {
	callID := r.PostFormValue("call")
	if callID == "" {
		callID = basen.NewUUIDBase62()
	}
	url := mux.BaseURL + "/call/" + url.PathEscape(callID)
	http.Redirect(w, r, url, 302)
}

func (mux *Mux) routeIndex(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
	return "index.html", nil, nil
}

func (mux *Mux) routeCall(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
	callID := url.PathEscape(path.Base(r.URL.Path))
	userID := basen.NewUUIDBase62()

	iceServers := iceauth.GetICEServers(mux.iceServers)
	iceServersJSON, _ := json.Marshal(iceServers)

	data := map[string]interface{}{
		"CallID":     callID,
		"UserID":     userID,
		"ICEServers": template.HTML(iceServersJSON),
	}
	return "call.html", data, nil
}
