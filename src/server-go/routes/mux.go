package routes

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"path"

	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr"
	"github.com/jeremija/peer-calls/src/server-go/basen"
	"github.com/jeremija/peer-calls/src/server-go/config"
	"github.com/jeremija/peer-calls/src/server-go/iceauth"
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
	networkType config.NetworkType,
	iceServers []iceauth.ICEServer,
	rooms RoomManager,
	tracks TracksManager,
) *Mux {
	box := packr.NewBox("../templates")
	templates := render.ParseTemplates(box)
	renderer := render.NewRenderer(templates, baseURL, version)

	iceServersJSON, _ := json.Marshal(iceServers)

	handler := chi.NewRouter()
	mux := &Mux{
		BaseURL:    baseURL,
		handler:    handler,
		iceServers: string(iceServersJSON),
	}

	var root string
	if baseURL == "" {
		root = "/"
	} else {
		root = baseURL
	}

	wsHandler := newWebSocketHandler(
		networkType,
		wsserver.NewWSS(rooms),
		iceServers,
		tracks,
	)

	handler.Route(root, func(router chi.Router) {
		router.Get("/", renderer.Render(mux.routeIndex))
		router.Handle("/static/*", static(baseURL+"/static", "../../../build"))
		router.Handle("/res/*", static(baseURL+"/res", "../../../res"))
		router.Post("/call", mux.routeNewCall)
		router.Get("/call/{callID}", renderer.Render(mux.routeCall))

		router.Mount("/ws", wsHandler)
	})

	return mux
}

func newWebSocketHandler(
	networkType config.NetworkType,
	wss *wsserver.WSS,
	iceServers []iceauth.ICEServer,
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

func static(prefix string, path string) http.Handler {
	box := packr.NewBox(path)
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
	data := map[string]interface{}{
		"CallID":     callID,
		"UserID":     userID,
		"ICEServers": template.HTML(mux.iceServers),
	}
	return "call.html", data, nil
}
