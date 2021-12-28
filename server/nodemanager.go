package server

import (
	"net"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/sfu"
	"github.com/peer-calls/peer-calls/v4/server/udptransport2"
)

const (
	pingTimeout    = 3 * time.Second
	destroyTimeout = 10 * time.Second
)

type NodeManager struct {
	params           *NodeManagerParams
	wg               sync.WaitGroup
	mu               sync.Mutex
	transportManager *udptransport2.Manager
}

type NodeManagerParams struct {
	Log           logger.Logger
	RoomManager   *ChannelRoomManager
	TracksManager TracksManager
	ListenAddr    *net.UDPAddr
	Nodes         []*net.UDPAddr
}

func NewNodeManager(params NodeManagerParams) (*NodeManager, error) {
	params.Log = params.Log.WithNamespaceAppended("node_manager").WithCtx(logger.Ctx{
		"local_addr": params.ListenAddr,
	})

	conn, err := net.ListenUDP("udp", params.ListenAddr)
	if err != nil {
		return nil, errors.Annotatef(err, "listen udp: %s", params.ListenAddr)
	}

	mediaEngine := NewMediaEngine()

	interceptorRegistry, err := NewInterceptorRegistry(mediaEngine)
	if err != nil {
		params.Log.Error("New interceptor registry", errors.Trace(err), nil)
	}

	params.Log.Info("Listen on UDP", nil)

	transportManager := udptransport2.NewManager(udptransport2.ManagerParams{
		Conn:                conn,
		Log:                 params.Log,
		Clock:               clock.New(),
		PingTimeout:         pingTimeout,
		DestroyTimeout:      destroyTimeout,
		InterceptorRegistry: interceptorRegistry,
	})

	nm := &NodeManager{
		params:           &params,
		transportManager: transportManager,
	}

	for _, addr := range params.Nodes {
		log := params.Log.WithCtx(logger.Ctx{
			"remote_addr": addr,
		})

		log.Info("Configuring remote node", nil)

		getFactoryResponse := transportManager.GetFactory(addr)

		nm.wg.Add(1)

		go func() {
			defer nm.wg.Done()

			factory, err := (<-getFactoryResponse).Result()
			if err != nil {
				log.Error("Create transport factory", errors.Trace(err), nil)
				return
			}

			nm.handleTransportFactory(factory)

			// TODO attempt reconnect once the factory is Done (after ticker is
			// implemented.
		}()
	}

	go nm.startTransportEventLoop()
	go nm.startRoomEventLoop()

	return nm, nil
}

func (nm *NodeManager) startTransportEventLoop() {
	for factory := range nm.transportManager.FactoriesChannel() {
		nm.handleTransportFactory(factory)
	}
}

func (nm *NodeManager) handleTransportFactory(factory *udptransport2.Factory) {
	nm.wg.Add(1)

	go func() {
		defer nm.wg.Done()

		for transport := range factory.TransportsChannel() {
			if err := nm.handleTransport(transport); err != nil {
				nm.params.Log.Error("Handle transport", errors.Trace(err), logger.Ctx{
					"client_id": transport.ClientID(),
					"stream_id": transport.StreamID(),
				})
			}
		}
	}()
}

func (nm *NodeManager) handleTransport(transport *udptransport2.Transport) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	streamID := transport.StreamID()

	nm.params.Log.Info("Add transport", logger.Ctx{
		"stream_id": streamID,
		"client_id": transport.ClientID(),
	})

	ch, err := nm.params.TracksManager.Add(streamID, transport)
	if err != nil {
		transport.Close()
		return errors.Annotatef(err, "add transport: %s", streamID)
	}

	nm.wg.Add(1)

	go func() {
		// FIXME currently all transport subscribe to all published tracks.
		//
		// This is the first step in an attempt to communicate the changes in
		// published tracks without actually adding the tracks to the peer
		// connection and wasting data.
		//
		// The frontend would need to be updated to handle these events and
		// subscribe to interesting streams. The first version might simply
		// subscribe to all tracks.
		//
		// The server transport would also need to be updated to:
		// 1. Not sub to tracks until at least one track is added.
		// 2. Automatically unsub from tracks after the last track is removed.
		//
		// Additionally, something needs to be done to prevent duplicate tracks
		// when more than 2 server nodes are present. For example, if there were
		// 3 nodes with 1 peer connection connected to node A, it would be
		// redundant if both server transports from node A and node B both
		// advertised the tracks from the peer connection to node C.
		for pubTrackEvent := range ch {
			// If tracks stop being automatically added to all other transports,
			// the AddTrack/RemoveTrack methods could be called here to provide
			// track metadata to streamTransport.
			//
			// However, there is one big difference in how StreamTransport _should_
			// handle addition or removal of tracks when compared to
			// WebRTCTransport: StreamTransport should only advertise tracks to
			// but not actually subscribe to them to prevent unnecessary network
			// traffic, whereas WebRTC transport always already receives tracks
			// from the app clients (browsers).
			//
			// The question is: how should StreamTransport handle/receive
			// subscription requests?

			logCtx := logger.Ctx{
				"client_id":        pubTrackEvent.PubTrack.ClientID,
				"user_id":          pubTrackEvent.PubTrack.PeerID,
				"track_id":         pubTrackEvent.PubTrack.TrackID,
				"track_event_type": pubTrackEvent.Type,
			}

			if pubTrackEvent.PubTrack.ClientID.IsServer() {
				// Do not forward tracks from other server transports to this node;
				// only forward tracks from WebRTC connections connected directly to
				// this server.
				nm.params.Log.Info("Skipping track from other server transport", logCtx)

				continue
			}

			err := nm.params.TracksManager.Sub(sfu.SubParams{
				Room:        streamID,
				PubClientID: pubTrackEvent.PubTrack.ClientID,
				TrackID:     pubTrackEvent.PubTrack.TrackID,
				SubClientID: transport.ClientID(),
			})
			if err != nil {
				nm.params.Log.Error("Failed to subscribe server transport to pub track event", errors.Trace(err), logCtx)

				continue
			}

			nm.params.Log.Info("Subscribed server transport to pub track event", logCtx)
		}
	}()

	return nil
}

func (nm *NodeManager) startRoomEventLoop() {
	for {
		roomEvent, err := nm.params.RoomManager.AcceptEvent()
		if err != nil {
			nm.params.Log.Error("Accept room event", errors.Trace(err), nil)

			return
		}

		log := nm.params.Log.WithCtx(logger.Ctx{
			"room_id": roomEvent.RoomName,
		})

		switch roomEvent.Type {
		case RoomEventTypeAdd:
			// Create new transports once the room was created on this node (e.g.
			// someone has joined on this node and the room was just created).
			// Transports initiated by other nodes will be accepted.
			for _, factory := range nm.transportManager.Factories() {
				err := factory.CreateTransport(roomEvent.RoomName)
				if err != nil {
					log.Error("Create transport", errors.Trace(err), nil)

					continue
				}
			}
		case RoomEventTypeRemove:
			// No need to do anything special if the room closes. The server
			// transports be automatically closed once the final peer disconnects.
			//
			// TODO I think one extra thing would need to be handled:
			// 1. Peer X joins room A on node 1.
			//    This starts the server transport in room A on node 1.
			// 2. The server transport initiates a connection to node 2.
			//    A transport is accepted and a room A is created on node 2.
			// 3. Peer Y joins room A on node 2.
			// 4. Peer Y leaves.
			//    Server transport on node 2 is terminated.
			// 5. Peer Z joins room A on node 2.
			//    A server transport is initiated after termination on node 2.
			//    Server transport on node 1 did not know about the termination.
			//
			// There needs to be a way to get a list of all tracks after a reconnect.
			for _, factory := range nm.transportManager.Factories() {
				err := factory.CloseTransport(roomEvent.RoomName)
				if err != nil {
					log.Error("Close transport", errors.Trace(err), nil)

					continue
				}
			}
		}
	}
}

func (nm *NodeManager) Close() error {
	nm.params.RoomManager.Close()
	nm.transportManager.Close()

	nm.wg.Wait()

	return nil
}
