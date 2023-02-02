package server_test

import (
	"context"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/message"
	"github.com/peer-calls/peer-calls/v4/server/pionlogger"
	"github.com/peer-calls/peer-calls/v4/server/sfu"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
)

func setupSFUServer(rooms server.RoomManager, jitterBufferEnabled bool) (s *httptest.Server, url string) {
	log := test.NewLogger()

	handler := server.NewSFUHandler(
		log,
		server.NewWSS(log, rooms),
		[]server.ICEServer{},
		server.NetworkConfigSFU{},
		sfu.NewTracksManager(log, jitterBufferEnabled),
	)
	s = httptest.NewServer(handler)
	url = "ws" + strings.TrimPrefix(s.URL, "http") + "/ws/"
	return
}

func TestSFU_ConnectDisconnect(t *testing.T) {
	log := test.NewLogger()

	defer goleak.VerifyNone(t)
	newAdapter := server.NewAdapterFactory(log, server.StoreConfig{})
	defer newAdapter.Close()
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	server, url := setupSFUServer(rooms, false)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	wsc := mustDialWS(t, ctx, url)
	err := wsc.Close(websocket.StatusNormalClosure, "")
	require.Nil(t, err, "error closing client socket")
}

func waitForUsersEvent(t *testing.T, ctx context.Context, ch <-chan message.Message) {
	t.Helper()

	for {
		select {
		case msg := <-ch:
			// t.Log("1 msg.Type", msg)
			if msg.Type == message.TypeUsers {
				assert.Equal(t, roomName, msg.Room)
				assert.Equal(t, identifiers.ClientID("__SERVER__"), msg.Payload.Users.Initiator)
				assert.Equal(t, []identifiers.ClientID{"__SERVER__"}, msg.Payload.Users.PeerIDs)
				return
			}
		case <-ctx.Done():
			t.Errorf("context timeout: %s", ctx.Err())
			return
		}
	}
}

func startSignalling(
	t *testing.T,
	wsClient *server.Client,
	wsRecvCh <-chan message.Message,
	clientID identifiers.ClientID,
	signaller *server.Signaller,
) {
	t.Helper()
	signalChan := signaller.SignalChannel()

	go func() {
		for msg := range wsRecvCh {
			switch msg.Type {
			case message.TypeSignal:
				err := signaller.Signal(msg.Payload.Signal.Signal)
				require.NoError(t, err, "error in receiving signal payload: %w", err)
			case message.TypePubTrack:
				if msg.Payload.PubTrack.Type == transport.TrackEventTypeAdd {
					err := wsClient.Write(message.NewSubTrack(msg.Room, message.SubTrack{
						Type:        transport.TrackEventTypeSub,
						TrackID:     msg.Payload.PubTrack.TrackID,
						PubClientID: msg.Payload.PubTrack.PubClientID,
					}))
					require.NoError(t, err, "error sending sub_track event to ws: %w", err)
				}

			default:
				// fmt.Println("unhandled ws message", msg.Type, payload)
			}
		}
	}()

	go func() {
		for signal := range signalChan {
			userSignal := message.UserSignal{
				PeerID: clientID,
				Signal: signal,
			}

			err := wsClient.Write(message.NewSignal(roomName, userSignal))
			// Sometimes there are late signals created even after the test has
			// finished successfully, so ignore the errors, but log them.

			if err != nil {
				t.Log(fmt.Errorf("error sending signal to ws: %w", err))
			}
		}
	}()
}

type peerCtx struct {
	pc        *webrtc.PeerConnection
	signaller *server.Signaller
	wsClient  *server.Client
	msg       <-chan message.Message
	close     func() error
}

func createPeerConnection(t *testing.T, ctx context.Context, url string, clientID identifiers.ClientID) peerCtx {
	t.Helper()

	var peerCtx peerCtx

	var closer test.Closer

	peerCtx.close = closer.Close

	wsc := mustDialWS(t, ctx, url)
	closer.AddFuncErr(func() error {
		return wsc.Close(websocket.StatusNormalClosure, "")
	})

	wsClient := server.NewClientWithID(wsc, clientID)
	msgChan := wsClient.Messages()

	err := wsClient.Write(message.NewReady(roomName, message.Ready{
		Nickname: "some-user",
	}))
	require.NoError(t, err, "error sending ready message")

	waitForUsersEvent(t, ctx, msgChan)
	require.Nil(t, wsClient.Err())

	var mediaEngine webrtc.MediaEngine
	server.RegisterCodecs(&mediaEngine, codecs.NewRegistryDefault())

	log := test.NewLogger()

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(&mediaEngine),
		webrtc.WithSettingEngine(webrtc.SettingEngine{
			LoggerFactory: pionlogger.NewFactory(log),
		}),
	)

	peerCtx.pc, err = api.NewPeerConnection(webrtc.Configuration{})
	require.Nil(t, err, "error creating peer connection")

	peerCtx.signaller, err = server.NewSignaller(
		log,
		false,
		peerCtx.pc,
	)
	require.Nil(t, err, "error creating signaller")

	closer.AddFuncErr(peerCtx.signaller.Close) // also closes pc

	startSignalling(t, wsClient, msgChan, "__SERVER__", peerCtx.signaller)

	return peerCtx
}

func waitPeerConnected(t *testing.T, ctx context.Context, pc *webrtc.PeerConnection) {
	t.Helper()
	waitCh := make(chan struct{})
	// signaller and negotiator do not use this method. if they were, resetting
	// this listener could mess up their functionality. in the future, the whole
	// pc could be wrapped by signaller or some other higher-level struct
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateConnected {
			pc.OnConnectionStateChange(nil)
			close(waitCh)
		}
	})
	wait(t, ctx, waitCh)
}

func sendVideoUntilDone(t *testing.T, done <-chan struct{}, track *webrtc.TrackLocalStaticSample) {
	for {
		select {
		case <-time.After(20 * time.Millisecond):
			assert.NoError(t, track.WriteSample(media.Sample{Data: []byte{0x00}}))
		case <-done:
			return
		}
	}
}

func wait(t *testing.T, ctx context.Context, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		return
	case <-ctx.Done():
		t.Error("context cancelled:", ctx.Err())
	}
}

func TestSFU_PeerConnection(t *testing.T) {
	log := test.NewLogger()

	defer goleak.VerifyNone(t)

	newAdapter := server.NewAdapterFactory(log, server.StoreConfig{})
	defer newAdapter.Close()

	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	srv, wsBaseURL := setupSFUServer(rooms, false)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	peerCtx := createPeerConnection(t, ctx, wsBaseURL+roomName.String()+"/"+clientID.String(), clientID)
	defer peerCtx.close()

	waitPeerConnected(t, ctx, peerCtx.pc)
}

func TestSFU_PeerConnection_DuplicateClientID(t *testing.T) {
	log := test.NewLogger()

	defer goleak.VerifyNone(t)

	newAdapter := server.NewAdapterFactory(log, server.StoreConfig{})
	defer newAdapter.Close()

	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	srv, wsBaseURL := setupSFUServer(rooms, false)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	url := wsBaseURL + roomName.String() + "/" + clientID.String()

	peerCtx1 := createPeerConnection(t, ctx, url, clientID)
	defer peerCtx1.close()

	waitPeerConnected(t, ctx, peerCtx1.pc)

	wsc2 := mustDialWS(t, ctx, url)
	defer wsc2.Close(websocket.StatusNormalClosure, "")

	client2 := server.NewClientWithID(wsc2, clientID)

	err := client2.Write(message.NewReady(roomName, message.Ready{
		Nickname: "some-other-user",
	}))
	require.NoError(t, err, "error sending ready message")

	select {
	case msg, ok := <-client2.Messages():
		assert.False(t, ok, "expected ws connection to be closed, but got: %+v", msg)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for connection 2 to be closed.")
	}

	require.NotNil(t, client2.Err(), "expected a client error")
	assert.Contains(t, client2.Err().Error(), "duplicate client id")

	// assert.EqualErrorf(t, client2.Err(), "duplicate client id", "ha")
}

func TestSFU_OnTrack(t *testing.T) {
	log := test.NewLogger()

	defer func() {
		goleak.VerifyNone(t)
	}()
	adapterFactory := server.NewAdapterFactory(log, server.StoreConfig{})

	log = log.WithNamespaceAppended("test")

	defer adapterFactory.Close()

	numHangUps := int64(0)

	// allClientsHungUpCtx will be done after all clients hung up. This means
	// the server-side goroutines will be cleared up
	allClientsHungUpCtx, allClientsHungUp := context.WithCancel(context.Background())

	defer func() {
		log.Info("Waiting for all clients to hang up", nil)
		<-allClientsHungUpCtx.Done()
	}()

	// Wrap the adapter factory so we can spy the websocket events. We only want
	// the test to be done after all the server-side stuff has been cleaned up.
	// This is done after the HANG_UP events.
	afWrapper := newAdapterFactoryWrapper(
		adapterFactory,
		func(roomID identifiers.RoomID, adapter server.Adapter) {
			mockClient := newMockClientWriter("SPY", func(msg message.Message) error {
				if msg.Type == message.TypeHangUp {
					log.Info("Hang up event", logger.Ctx{
						"client_id": msg.Payload.HangUp.PeerID,
					})
					// We broadcast HANG_UP twice for each socket disconnected, once for
					// the socket, the other time for peer disconneting. Need see if
					// there's a way to do it only once. But the client-side can also
					// send the HANG_UP event without disconnecting.
					if h := atomic.AddInt64(&numHangUps, 1); h == 4 {
						allClientsHungUp()
						go adapter.Remove("SPY")
					}
				}

				return nil
			})

			adapter.Add(mockClient)
		},
	)

	rooms := server.NewAdapterRoomManager(afWrapper.NewAdapter)
	srv, wsBaseURL := setupSFUServer(rooms, false)

	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	peerCtx1 := createPeerConnection(t, ctx, wsBaseURL+roomName.String()+"/"+clientID.String(), clientID)
	defer peerCtx1.close()

	waitPeerConnected(t, ctx, peerCtx1.pc)

	fmt.Println("PEER 1 CONNECTED")

	peerCtx2 := createPeerConnection(t, ctx, wsBaseURL+roomName.String()+"/"+clientID2.String(), clientID2)
	defer peerCtx2.close()

	waitPeerConnected(t, ctx, peerCtx2.pc)

	fmt.Println("PEER 2 CONNECTED")

	pc1 := peerCtx1.pc
	pc2 := peerCtx2.pc

	onTrackFired, onTrackFiredDone := context.WithCancel(ctx)
	onTrackEOF, onTrackEOFDone := context.WithCancel(ctx)
	pc2.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log := log.WithCtx(logger.Ctx{"ssrc": remoteTrack.SSRC()})

		log.Info("OnTrack", nil)

		onTrackFiredDone()
		for {
			_, _, err := remoteTrack.ReadRTP()
			fmt.Println("got rtp", err)
			if err != nil {
				log.Info("pc2 remote track ended", nil)
				assert.Equal(t, io.EOF, err, "error reading track")
				onTrackEOFDone()
				return
			}
		}
	})

	localTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: "video/VP8"}, "track-one", "stream-one",
	)
	require.NoError(t, err)

	log.Info("AddTrack start", logger.Ctx{
		"track_id":  localTrack.ID(),
		"stream_id": localTrack.StreamID(),
	})

	sender, err := pc1.AddTrack(localTrack)

	log.Info("AddTrack end", logger.Ctx{
		"track_id":  localTrack.ID(),
		"stream_id": localTrack.StreamID(),
	})

	require.NoError(t, err)
	require.NotNil(t, sender)

	go sendVideoUntilDone(t, onTrackFired.Done(), localTrack)

	log.Info("sending negotiate request (1)", nil)
	wait(t, ctx, peerCtx1.signaller.Negotiate())

	log.Info("waiting for track", nil)

	<-onTrackFired.Done()
	assert.Equal(t, context.Canceled, onTrackFired.Err(), "test timed out")

	log.Info("got track", nil)

	// trigger io.EOF when reading from track2
	assert.NoError(t, pc1.RemoveTrack(sender))

	log.Info("sending negotiate request (2)", nil)
	wait(t, ctx, peerCtx1.signaller.Negotiate())

	<-onTrackEOF.Done()
	assert.Equal(t, context.Canceled, onTrackEOF.Err(), "test timed out")

	log.Info("waiting for peer2 negotiation to be done (3)", nil)
	// server will want to negotiate after track is removed so we wait for negotiation to complete
	wait(t, ctx, peerCtx2.signaller.NegotiationDone())
}

type mockClientWriter struct {
	clientID identifiers.ClientID

	mu        sync.Mutex
	metadata  string
	onMessage func(message.Message) error
}

func newMockClientWriter(
	clientID identifiers.ClientID,
	onMessage func(message.Message) error,
) *mockClientWriter {
	return &mockClientWriter{
		clientID:  clientID,
		onMessage: onMessage,
	}
}

func (m *mockClientWriter) ID() identifiers.ClientID {
	return m.clientID
}

func (m *mockClientWriter) Write(msg message.Message) error {
	return errors.Trace(m.onMessage(msg))
}

func (m *mockClientWriter) Metadata() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.metadata
}

func (m *mockClientWriter) SetMetadata(metadata string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata = metadata
}

type adapterFactoryWrapper struct {
	adapterFactory *server.AdapterFactory
	onNewAdapter   func(roomID identifiers.RoomID, adapter server.Adapter)
}

func newAdapterFactoryWrapper(
	adapterFactory *server.AdapterFactory,
	onNewAdapter func(roomID identifiers.RoomID, adapter server.Adapter),
) *adapterFactoryWrapper {
	return &adapterFactoryWrapper{
		adapterFactory: adapterFactory,
		onNewAdapter:   onNewAdapter,
	}
}

func (m *adapterFactoryWrapper) NewAdapter(roomID identifiers.RoomID) server.Adapter {
	adapter := m.adapterFactory.NewAdapter(roomID)

	m.onNewAdapter(roomID, adapter)

	return adapter
}
