package server

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/go-chi/chi"
	"github.com/gobuffalo/packr"
	"github.com/pion/webrtc/v2"
)

type Mux struct {
	BaseURL    string
	handler    *chi.Mux
	iceServers []ICEServer
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.handler.ServeHTTP(w, r)
}

type TracksManager interface {
	Add(room string, clientID string, pc *webrtc.PeerConnection, dc *webrtc.DataChannel, s *Signaller)
	GetTracksMetadata(clientID string) ([]TrackMetadata, bool)
}

type RoomManager interface {
	Enter(room string) Adapter
	Exit(room string)
}

func NewMux(
	loggerFactory LoggerFactory,
	baseURL string,
	version string,
	network NetworkConfig,
	iceServers []ICEServer,
	rooms RoomManager,
	tracks TracksManager,
) *Mux {
	box := packr.NewBox("./templates")
	templates := ParseTemplates(box)
	renderer := NewRenderer(loggerFactory, templates, baseURL, version)

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
		loggerFactory,
		network,
		NewWSS(loggerFactory, rooms),
		iceServers,
		tracks,
	)

	handler.Route(root, func(router chi.Router) {
		router.Get("/", renderer.Render(mux.routeIndex))
		router.Handle("/static/*", static(baseURL+"/static", packr.NewBox("../build")))
		router.Handle("/res/*", static(baseURL+"/res", packr.NewBox("../res")))
		router.Post("/call", mux.routeNewCall)
		router.Get("/call/{callID}", renderer.Render(mux.routeCall))

		router.Mount("/ws", wsHandler)
	})

	return mux
}

func newWebSocketHandler(
	loggerFactory LoggerFactory,
	network NetworkConfig,
	wss *WSS,
	iceServers []ICEServer,
	tracks TracksManager,
) http.Handler {
	switch network.Type {
	case NetworkTypeSFU:
		log.Println("Using network type sfu")
		return NewSFUHandler(loggerFactory, wss, iceServers, network.SFU, tracks)
	default:
		log.Println("Using network type mesh")
		return NewMeshHandler(loggerFactory, wss)
	}
}

func static(prefix string, box packr.Box) http.Handler {
	fileServer := http.FileServer(http.FileSystem(box))
	return http.StripPrefix(prefix, fileServer)
}

func (mux *Mux) routeNewCall(w http.ResponseWriter, r *http.Request) {
	callID := r.PostFormValue("call")
	if callID == "" {
		callID = NewUUIDBase62()
	}
	url := mux.BaseURL + "/call/" + url.PathEscape(callID)
	http.Redirect(w, r, url, 302)
}

func (mux *Mux) routeIndex(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
	return "index.html", nil, nil
}

func (mux *Mux) routeCall(w http.ResponseWriter, r *http.Request) (string, interface{}, error) {
	callID := url.PathEscape(path.Base(r.URL.Path))
	userID := NewUUIDBase62()

	iceServers := GetICEAuthServers(mux.iceServers)
	iceServersJSON, _ := json.Marshal(iceServers)

	data := map[string]interface{}{
		"Nickname":   r.Header.Get("X-Forwarded-User"),
		"CallID":     callID,
		"UserID":     userID,
		"ICEServers": template.HTML(iceServersJSON),
	}
	return "call.html", data, nil
}
