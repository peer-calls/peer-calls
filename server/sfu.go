package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/identifiers"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/message"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
)

const IOSH264Fmtp = "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f"

const localPeerID identifiers.ClientID = "__SERVER__"

const serverIsInitiator = true

func NewSFUHandler(
	log logger.Logger,
	wss *WSS,
	iceServers []ICEServer,
	sfuConfig NetworkConfigSFU,
	tracksManager TracksManager,
) *SFU {
	log = log.WithNamespaceAppended("sfu")

	webRTCTransportFactory := NewWebRTCTransportFactory(log, iceServers, sfuConfig)

	return &SFU{log, wss, tracksManager, webRTCTransportFactory}
}

type SFU struct {
	log           logger.Logger
	wss           *WSS
	tracksManager TracksManager

	webRTCTransportFactory *WebRTCTransportFactory
}

func (sfu *SFU) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sub, err := sfu.wss.Subscribe(w, r)
	if err != nil {
		sfu.log.Error("Accept websocket connection", errors.Trace(err), nil)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log := sfu.log.WithCtx(logger.Ctx{
		"room_id":   sub.Room,
		"client_id": sub.ClientID,
	})

	socketHandler := NewSocketHandler(
		log,
		sfu.tracksManager,
		sfu.webRTCTransportFactory,
		sub.ClientID,
		sub.Room,
		sub.Adapter,
	)

	for message := range sub.Messages {
		err := socketHandler.HandleMessage(message)
		if err != nil {
			log.Error("Handle websocket message", errors.Trace(err), nil)
		}
	}

	socketHandler.Cleanup()
}

type SocketHandler struct {
	log                    logger.Logger
	tracksManager          TracksManager
	webRTCTransportFactory *WebRTCTransportFactory
	webRTCTransport        *WebRTCTransport
	adapter                Adapter
	clientID               identifiers.ClientID
	room                   identifiers.RoomID

	mu sync.Mutex
}

func NewSocketHandler(
	log logger.Logger,
	tracksManager TracksManager,
	webRTCTransportFactory *WebRTCTransportFactory,
	clientID identifiers.ClientID,
	room identifiers.RoomID,
	adapter Adapter,
) *SocketHandler {
	return &SocketHandler{
		log:                    log.WithNamespaceAppended("sfu"),
		tracksManager:          tracksManager,
		webRTCTransportFactory: webRTCTransportFactory,
		clientID:               clientID,
		room:                   room,
		adapter:                adapter,
	}
}

func (sh *SocketHandler) HandleMessage(msg message.Message) error {
	sh.mu.Lock()
	defer sh.mu.Unlock()

	var err error

	switch msg.Type {
	case message.TypeHangUp:
		err = errors.Trace(sh.handleHangUp(*msg.Payload.HangUp))
	case message.TypeReady:
		err = errors.Trace(sh.handleReady(*msg.Payload.Ready))
	case message.TypeSignal:
		err = errors.Trace(sh.handleSignal(*msg.Payload.Signal))
	case message.TypeSubTrack:
		err = errors.Trace(sh.handleSubTrackEvent(*msg.Payload.SubTrack))
	case message.TypePing:
	default:
		err = errors.Errorf("Unhandled event: %+v", msg)
	}

	return errors.Trace(err)
}

func (sh *SocketHandler) Cleanup() {
	if sh.webRTCTransport != nil {
		if err := sh.webRTCTransport.Close(); err != nil {
			sh.log.Error("Cleanup: close WebRTCTransport", errors.Trace(err), nil)
		}
	}

	err := sh.adapter.Broadcast(
		message.NewHangUp(sh.room, message.HangUp{
			PeerID: sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Error("Cleanup: broadcast hangUp", errors.Trace(err), nil)
	}
}

func (sh *SocketHandler) handleSubTrackEvent(sub message.SubTrack) error {
	var err error

	switch sub.Type {
	case transport.TrackEventTypeSub:
		err = sh.tracksManager.Sub(sfu.SubParams{
			PubClientID: sub.PubClientID,
			Room:        sh.room,
			TrackID:     sub.TrackID,
			SubClientID: sh.clientID,
		})
		err = errors.Trace(err)
	case transport.TrackEventTypeUnsub:
		err = sh.tracksManager.Unsub(sfu.SubParams{
			PubClientID: sub.PubClientID,
			Room:        sh.room,
			TrackID:     sub.TrackID,
			SubClientID: sh.clientID,
		})
		err = errors.Trace(err)
	default:
		err = errors.Errorf("invalid sub track event: %+v", sub)
	}

	return errors.Trace(err)
}

func (sh *SocketHandler) handleHangUp(_ message.HangUp) error {
	clientID := sh.clientID

	sh.log.Info("hangUp event", nil)

	if sh.webRTCTransport != nil {
		err := sh.webRTCTransport.Close()
		if err != nil {
			return errors.Annotatef(err, "hangUp: error closing peer connection for client: %s", clientID)
		}
	}

	return nil
}

func (sh *SocketHandler) handleReady(msg message.Ready) error {
	adapter := sh.adapter
	roomID := sh.room
	clientID := sh.clientID
	// peerID is the same as clientID for webrtc connections.
	peerID := identifiers.PeerID(sh.clientID)

	initiator := localPeerID
	if !serverIsInitiator {
		initiator = clientID
	}

	start := time.Now()

	sh.log.Info("ready event", logger.Ctx{
		"initiator": initiator,
	})

	if sh.webRTCTransport != nil {
		return errors.Errorf("unexpected ready event in room %s - already have a webrtc transport", roomID)
	}

	adapter.SetMetadata(clientID, msg.Nickname)

	clients, err := getReadyClients(adapter)
	if err != nil {
		return errors.Annotatef(err, "get ready clients")
	}

	err = adapter.Broadcast(
		message.NewUsers(roomID, message.Users{
			Initiator: initiator,
			PeerIDs:   []identifiers.ClientID{localPeerID},
			Nicknames: clients,
		}),
	)
	if err != nil {
		return errors.Annotatef(err, "broadcasting users")
	}

	webRTCTransport, err := sh.webRTCTransportFactory.NewWebRTCTransport(roomID, clientID, peerID)
	if err != nil {
		return errors.Annotatef(err, "create new WebRTCTransport")
	}

	sh.webRTCTransport = webRTCTransport

	prometheusWebRTCConnTotal.Inc()
	prometheusWebRTCConnActive.Inc()

	pubTrackEventsCh, err := sh.tracksManager.Add(roomID, webRTCTransport)
	if err != nil {
		return errors.Trace(err)
	}

	go func() {
		for pubTrackEvent := range pubTrackEventsCh {
			err := sh.adapter.Emit(clientID, message.NewPubTrack(roomID, message.PubTrack{
				PubClientID: pubTrackEvent.PubTrack.ClientID,
				TrackID:     pubTrackEvent.PubTrack.TrackID,
				PeerID:      pubTrackEvent.PubTrack.PeerID,
				Kind:        pubTrackEvent.PubTrack.Kind,
				Type:        pubTrackEvent.Type,
			}))
			if err != nil {
				sh.log.Error("Emit pub track event", errors.Trace(err), nil)
			}
		}
	}()

	go sh.processLocalSignals(webRTCTransport.SignalChannel(), start)

	return nil
}

func (sh *SocketHandler) handleSignal(signal message.UserSignal) error {
	if sh.webRTCTransport == nil {
		return errors.Errorf("signal: webRTCTransport not initialized")
	}

	err := sh.webRTCTransport.Signal(signal.Signal)
	return errors.Annotate(err, "handleSignal")
}

func (sh *SocketHandler) processLocalSignals(signals <-chan message.Signal, startTime time.Time) {
	room := sh.room
	adapter := sh.adapter
	clientID := sh.clientID

	for signal := range signals {
		userSignal := message.UserSignal{
			PeerID: localPeerID,
			Signal: signal,
		}

		err := adapter.Emit(clientID, message.NewSignal(room, userSignal))
		if err != nil {
			sh.log.Error("Send local signal", errors.Trace(err), nil)
			// TODO abort connection
		}
	}

	prometheusWebRTCConnActive.Dec()
	prometheusWebRTCConnDuration.Observe(time.Since(startTime).Seconds())

	sh.mu.Lock()
	defer sh.mu.Unlock()
	sh.webRTCTransport = nil
	sh.log.Info("Peer connection closed, send hangUp event", nil)
	adapter.SetMetadata(clientID, "")

	err := sh.adapter.Broadcast(
		message.NewHangUp(room, message.HangUp{
			PeerID: sh.clientID,
		}),
	)
	if err != nil {
		sh.log.Error("Broadcast hangUp", errors.Trace(err), nil)
	}
}
