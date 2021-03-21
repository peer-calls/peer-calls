package sfu

import (
	"io"
	"strings"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/multierr"
	"github.com/peer-calls/peer-calls/server/pubsub"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type PeerManager struct {
	log logger.Logger
	mu  sync.RWMutex
	wg  sync.WaitGroup

	jitterHandler          JitterHandler
	trackBitrateEstimators *TrackBitrateEstimators

	// webrtcTransports indexed by ClientID
	webrtcTransports map[string]transport.Transport
	serverTransports map[string]transport.Transport

	room string

	// // pubsub keeps track of published tracks and its subscribers.
	pubsub *pubsub.PubSub
}

func NewPeerManager(room string, log logger.Logger, jitterHandler JitterHandler) *PeerManager {
	return &PeerManager{
		log:                    log.WithNamespaceAppended("room_peers_manager"),
		webrtcTransports:       map[string]transport.Transport{},
		serverTransports:       map[string]transport.Transport{},
		jitterHandler:          jitterHandler,
		trackBitrateEstimators: NewTrackBitrateEstimators(),
		room:                   room,

		pubsub: pubsub.New(log),
	}
}

// func (t *PeerManager) addTrack(clientID string, track transport.TrackRemote) {
// 	t.mu.Lock()
// 	defer t.mu.Unlock()

// 	log := t.log.WithCtx(logger.Ctx{
// 		"client_id": clientID,
// 		"track_id":  track.Track().UniqueID(),
// 	})

// 	log.Trace("Add track", logger.Ctx{
// 		"track": track,
// 	})

// 	// t.pubsub.Pub(clientID, track)

// 	// FIXME pion3 update
// 	// // Let the server transports know of the new track.
// 	// for subClientID, subTransport := range t.serverTransports {
// 	// 	if subClientID != clientID {
// 	// 		// Note: pubsub.Sub is _not_ called here because the server transport
// 	// 		// does not want to receive RTP/RTCP data immmediatelly if there are
// 	// 		// no interested parties on the other end of the connection. This is done
// 	// 		// later, when Pub/Sub events are handled. These events are sent thorugh
// 	// 		// servertransport.MetadataTransport - see the goroutine reading from
// 	// 		// TrackEventsChannel for more info.
// 	// 		if track_, err := subTransport.AddTrack(track.Track()); err != nil {
// 	// 			log.Error("Add track", errors.Trace(err), logger.Ctx{
// 	// 				"sub_client_id": subClientID,
// 	// 			})

// 	// 			continue
// 	// 		}
// 	// 	}
// 	// }
// }

func (t *PeerManager) broadcast(clientID string, msg webrtc.DataChannelMessage) {
	t.mu.Lock()
	defer t.mu.Unlock()

	broadcast := func(tr transport.Transport) {
		if otherClientID := tr.ClientID(); otherClientID != clientID {
			// FIXME async
			err := <-tr.Send(msg)
			if err != nil {
				t.log.Error("Broadcast", errors.Trace(err), logger.Ctx{
					"client_id":       clientID,
					"other_client_id": otherClientID,
				})
			}
		}
	}

	for _, tr := range t.webrtcTransports {
		broadcast(tr)
	}

	for _, tr := range t.serverTransports {
		broadcast(tr)
	}
}

// func (t *PeerManager) getTransportBySSRC(subClientID string, ssrc uint32) (
// 	transport transport.Transport, ok bool,
// ) {
// 	t.mu.Lock()
// 	defer t.mu.Unlock()

// 	clientID, ok := t.pubsub.PubClientID(subClientID, ssrc)
// 	if !ok {
// 		return nil, false
// 	}

// 	transport, ok = t.getTransport(clientID)

// 	return transport, ok
// }

func (t *PeerManager) getTransport(clientID string) (transport.Transport, bool) {
	transport, ok := t.webrtcTransports[clientID]
	if !ok {
		transport, ok = t.serverTransports[clientID]
	}

	return transport, ok
}

func (t *PeerManager) Add(tr transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	// log := t.log.WithCtx(logger.Ctx{
	// 	"client_id": tr.ClientID(),
	// })

	pubTrackEventSub, err := t.pubsub.SubscribeToEvents(tr.ClientID())
	if err != nil {
		return nil, errors.Annotatef(err, "subscribe to events: %s", tr.ClientID())
	}

	pubTracks := t.pubsub.Tracks()

	pubTrackEventsCh := make(chan pubsub.PubTrackEvent)

	t.wg.Add(1)

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		defer close(pubTrackEventsCh)

		for _, pubTrack := range pubTracks {
			if pubTrack.ClientID != tr.ClientID() {
				pubTrackEventsCh <- pubsub.PubTrackEvent{
					PubTrack: pubsub.PubTrack{
						ClientID: pubTrack.ClientID,
						UserID:   pubTrack.UserID,
						TrackID:  pubTrack.TrackID,
					},
					Type: transport.TrackEventTypeAdd,
				}
			}
		}

		for event := range pubTrackEventSub {
			if event.PubTrack.ClientID != tr.ClientID() {
				pubTrackEventsCh <- event
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for remoteTrack := range tr.RemoteTracksChannel() {
			remoteTrack := remoteTrack

			t.pubsub.Pub(tr.ClientID(), pubsub.NewTrackReader(remoteTrack, func() {
				t.mu.Lock()

				t.pubsub.Unpub(tr.ClientID(), remoteTrack.Track().UniqueID())

				t.mu.Unlock()
			}))
			// switch trackEvent.Type {
			// case transport.TrackEventTypeAdd:
			// 	t.addTrack(tr.ClientID(), trackEvent.TrackInfo.Track)
			// case transport.TrackEventTypeRemove:
			// 	t.removeTrack(tr.ClientID(), trackEvent.TrackInfo.Track)
			// The following events are generated only by server transport.
			// FIXME pion3: disabled for now
			// case transport.TrackEventTypeSub:
			// 	if err := t.Sub(SubParams{
			// 		Room:        t.room,
			// 		PubClientID: trackEvent.TrackInfo.Track.(*servertransport.ServerTrack).UserID(),
			// 		TrackID:     trackEvent.TrackInfo.Track.UniqueID(),
			// 		SubClientID: tr.ClientID(),
			// 	}); err != nil {
			// 		log.Error("sub failed", errors.Trace(err), nil)
			// 	}
			// case transport.TrackEventTypeUnsub:
			// 	if err := t.Unsub(SubParams{
			// 		Room:        t.room,
			// 		PubClientID: trackEvent.TrackInfo.Track.(*servertransport.ServerTrack).UserID(),
			// 		TrackID:     trackEvent.TrackInfo.Track.UniqueID(),
			// 		SubClientID: tr.ClientID(),
			// 	}); err != nil {
			// 		log.Error("sub failed", errors.Trace(err), nil)
			// 	}
			// }
		}
	}()

	// t.wg.Add(1)
	//
	// go func() {
	// 	defer t.wg.Done()
	//
	// 	for packet := range tr.RTPChannel() {
	// 		rtcpPacket := t.jitterHandler.HandleRTP(packet)
	// 		if rtcpPacket != nil {
	// 			err := tr.WriteRTCP([]rtcp.Packet{rtcpPacket})
	// 			if err != nil {
	// 				log.Error("WriteRTCP", errors.Trace(err), nil)
	// 			}
	// 		}
	//
	// 		t.mu.Lock()
	//
	// 		subTransports := t.pubsub.Subscribers(tr.ClientID(), packet.SSRC)
	//
	// 		t.mu.Unlock()
	//
	// 		for subClientID, subTransport := range subTransports {
	// 			if _, err := subTransport.(transport.Transport).WriteRTP(packet); err != nil {
	// 				log.Error("WriteRTP", errors.Trace(err), logger.Ctx{
	// 					"pub_client_id": tr.ClientID(),
	// 					"sub_client_id": subClientID,
	// 					"ssrc":          packet.SSRC,
	// 				})
	// 			}
	// 		}
	// 	}
	// }()
	//
	// t.wg.Add(1)
	//
	// go func() {
	//	defer t.wg.Done()
	//
	//	handleREMB := func(packet *rtcp.ReceiverEstimatedMaximumBitrate) error {
	//		errs := multierr.New()
	//
	//		bitrate := t.trackBitrateEstimators.Estimate(tr.ClientID(), packet.SSRCs, packet.Bitrate)
	//		packet.Bitrate = bitrate
	//
	//		transportsSet := map[transport.Transport]struct{}{}
	//
	//		for _, ssrc := range packet.SSRCs {
	//			sourceTransport, ok := t.getTransportBySSRC(tr.ClientID(), ssrc)
	//			if ok {
	//				transportsSet[sourceTransport] = struct{}{}
	//			}
	//		}
	//
	//		for sourceTransport := range transportsSet {
	//			err := sourceTransport.WriteRTCP([]rtcp.Packet{packet})
	//			errs.Add(errors.Trace(err))
	//		}
	//
	//		return errors.Annotatef(errs.Err(), "remb")
	//	}
	//
	//	handlePLI := func(packet *rtcp.PictureLossIndication) error {
	//		sourceTransport, ok := t.getTransportBySSRC(tr.ClientID(), packet.MediaSSRC)
	//		if !ok {
	//			return errors.Errorf("no source transport for PictureLossIndication for track: %d", packet.MediaSSRC)
	//		}
	//
	//		err := sourceTransport.WriteRTCP([]rtcp.Packet{packet})
	//
	//		return errors.Annotate(err, "write rtcp")
	//	}
	//
	//	handleNack := func(packet *rtcp.TransportLayerNack) error {
	//		errs := multierr.New()
	//
	//		foundRTPPackets, nack := t.jitterHandler.HandleNack(packet)
	//		for _, rtpPacket := range foundRTPPackets {
	//			if _, err := tr.WriteRTP(rtpPacket); err != nil {
	//				errs.Add(errors.Annotate(err, "write rtp"))
	//			}
	//		}
	//
	//		if nack != nil {
	//			sourceTransport, ok := t.getTransportBySSRC(tr.ClientID(), packet.MediaSSRC)
	//			if ok {
	//				if err := sourceTransport.WriteRTCP([]rtcp.Packet{nack}); err != nil {
	//					errs.Add(errors.Annotate(err, "write rtcp"))
	//				}
	//			}
	//		}
	//
	//		return errors.Annotatef(errs.Err(), "nack")
	//	}
	//
	//	for pkts := range tr.RTCPChannel() {
	//		for _, pkt := range pkts {
	//			var err error
	//			switch packet := pkt.(type) {
	//			case *rtcp.ReceiverEstimatedMaximumBitrate:
	//				err = errors.Trace(handleREMB(packet))
	//			case *rtcp.PictureLossIndication:
	//				err = errors.Trace(handlePLI(packet))
	//			case *rtcp.TransportLayerNack:
	//				err = errors.Trace(handleNack(packet))
	//			case *rtcp.SourceDescription:
	//			case *rtcp.ReceiverReport:
	//				// ReceiverReport is sent by remote side when it sent no packets
	//				// (since the last report?).
	//				//
	//				// The reception reports in this packet are about local tracks being
	//				// sent to the remote side of this transport.
	//			case *rtcp.SenderReport:
	//				// The sender report is about tracks currently being received from
	//				// the remote side of this transport.
	//				//
	//				// The reception reports in this packet are about local tracks being
	//				// sent to the remote side of this transport.
	//			default:
	//				log.Error(fmt.Sprintf("Unhandled RTCP Packet: %T", pkt), nil, logger.Ctx{
	//					"destination_ssrc": pkt.DestinationSSRC(),
	//				})
	//			}
	//
	//			if err != nil {
	//				// Log error and do not return early because the RTCP channel still
	//				// needs to be emptied.
	//				log.Error("Send RTCP to source peer", errors.Trace(err), nil)
	//			}
	//		}
	//	}
	//}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for msg := range tr.MessagesChannel() {
			t.broadcast(tr.ClientID(), msg)
		}
	}()

	t.wg.Done()

	t.mu.Lock()
	defer t.mu.Unlock()

	switch tr.Type() {
	case transport.TypeServer:
		t.serverTransports[tr.ClientID()] = tr

		// FIXME pion3 upgrade let the servers know of the new tracks.
		// for _, pubTransport := range t.webrtcTransports {
		// 	for _, trackInfo := range pubTransport.RemoteTracks() {
		// 		// FIXME should this be tr.AddTrack???
		// 		if err := pubTransport.AddTrack(trackInfo.Track); err != nil {
		// 			log.Error("add track", errors.Trace(err), logger.Ctx{
		// 				"pub_client_id": pubTransport.ClientID(),
		// 				"sub_client_id": tr.ClientID(),
		// 				"track_id":      trackInfo.Track.UniqueID(),
		// 			})
		// 		}
		// 	}
		// }

	case transport.TypeWebRTC:
		t.webrtcTransports[tr.ClientID()] = tr
	}

	return pubTrackEventsCh, nil
}

func (t *PeerManager) Sub(params SubParams) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	tr, ok := t.getTransport(params.SubClientID)
	if !ok {
		return errors.Errorf("transport not found: %s", params.PubClientID)
	}

	sender, err := t.pubsub.Sub(params.PubClientID, params.TrackID, tr)
	if err != nil {
		return errors.Trace(err)
	}

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		logCtx := logger.Ctx{
			"pub_client_id": params.PubClientID,
			"track_id":      params.TrackID,
			"sub_client_id": params.SubClientID,
		}

		getTrackProps := func(trackID transport.TrackID) (pubsub.TrackProps, bool) {
			t.mu.Lock()

			props, ok := t.pubsub.TrackPropsByTrackID(trackID)

			t.mu.Unlock()

			return props, ok
		}

		handlePacket := func(packet rtcp.Packet) error {
			// NOTE: REMB and NACK are now handled by pion/webrtc interceptors so we
			// don't have to explicitly handle them here.
			switch pkt := packet.(type) {
			// PLI cannot be handled by interceptors since it's implementation
			// specific. We need to find the source and send the PLI packet. We also
			// need to make sure to set the correct SSRC before the packet is
			// forwarded, since pion/webrtc/v3 no longer uses the same SSRCs between
			// different peer connections.
			case *rtcp.PictureLossIndication:
				props, ok := getTrackProps(params.TrackID)
				if !ok {
					return errors.Annotatef(pubsub.ErrTrackNotFound, "got RTCP for track that was not found")
				}

				transport, ok := t.getTransport(props.ClientID)
				if !ok {
					return errors.Errorf("transport not found: %s", props.ClientID)
				}

				// Important: set the correct SSRC before sending the packet to source.
				pkt.MediaSSRC = uint32(props.SSRC)
				pkt.SenderSSRC = uint32(props.SSRC)

				if err := transport.WriteRTCP([]rtcp.Packet{pkt}); err != nil {
					return errors.Annotatef(err, "sending PLI back to source: %s", props.ClientID)
				}

				// TODO remove this log.
				t.log.Info("Sent PLI back to source", logCtx)
			default:
			}

			return nil
		}

		for {
			packets, _, err := sender.ReadRTCP()
			if err != nil {
				if !multierr.Is(err, io.EOF) {
					t.log.Error("Read RTCP for sender", errors.Trace(err), logCtx)
				}

				return
			}

			for _, packet := range packets {
				if err := handlePacket(packet); err != nil {
					t.log.Error("Handling RTCP packet", errors.Trace(err), logCtx)
				}
			}
		}
	}()

	return nil
}

func (t *PeerManager) Unsub(params SubParams) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	err := t.pubsub.Unsub(params.PubClientID, params.TrackID, params.SubClientID)

	return errors.Trace(err)
}

// TracksMetadata retrieves local track metadata for a specific transport.
func (t *PeerManager) TracksMetadata(clientID string) (m []TrackMetadata, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tr, ok := t.getTransport(clientID)
	if !ok {
		return m, false
	}

	tracks := tr.LocalTracks()
	m = make([]TrackMetadata, 0, len(tracks))

	for _, trackInfo := range tracks {
		track := trackInfo.Track

		var kind webrtc.RTPCodecType

		codec := track.Codec()

		switch {
		case strings.HasPrefix(codec.MimeType, "audio/"):
			kind = webrtc.RTPCodecTypeAudio
		case strings.HasPrefix(codec.MimeType, "video/"):
			kind = webrtc.RTPCodecTypeVideo
		default:
			kind = webrtc.RTPCodecType(0)
		}

		trackMetadata := TrackMetadata{
			Mid:      trackInfo.MID(),
			StreamID: track.StreamID(),
			UserID:   track.UserID(),
			Kind:     kind.String(),
		}

		t.log.Trace("GetTracksMetadata", logger.Ctx{
			"track_id":  track.UniqueID(),
			"client_id": clientID,
		})

		m = append(m, trackMetadata)
	}

	return m, true
}

// Remove removes the transport and unsubscribes it from track events.
func (t *PeerManager) Remove(clientID string) {
	t.log.Trace("Remove", logger.Ctx{
		"client_id": clientID,
	})

	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.pubsub.UnsubscribeFromEvents(clientID); err != nil {
		t.log.Error("Unsubscribe from events", errors.Trace(err), logger.Ctx{
			"client_id": clientID,
		})
	}

	t.pubsub.Terminate(clientID)

	if _, ok := t.serverTransports[clientID]; ok {
		// WebRTC transports do not need to be explicitly terminated, only
		// ServerTransports do. This is because a closed WebRTC tranports will
		// still dispatch track remove events after the streams are closed.

		// FIXME maybe they do!
		// t.pubsub.Terminate(clientID)
		delete(t.serverTransports, clientID)
	} else {
		delete(t.webrtcTransports, clientID)
	}

	t.trackBitrateEstimators.RemoveReceiverEstimations(clientID)
}

// func (t *PeerManager) removeTrack(clientID string, track transport.Track) {
// 	trackID := track.UniqueID()

// 	t.log.Trace("Remove track", logger.Ctx{
// 		"client_id": clientID,
// 		"track_id":  trackID,
// 	})

// 	t.mu.Lock()
// 	defer t.mu.Unlock()

// 	t.pubsub.Unpub(clientID, trackID)

// 	// FIXME re-enable REMB
// 	// t.trackBitrateEstimators.Remove(ssrc)

// 	// Let the server transports know the track has been removed.
// 	for subClientID, subTransport := range t.serverTransports {
// 		if subClientID != clientID {
// 			if err := subTransport.RemoveTrack(trackID); err != nil {
// 				t.log.Error("Remove track", errors.Trace(err), logger.Ctx{
// 					"pub_client_id": clientID,
// 					"sub_client_id": subClientID,
// 					"track_id":      trackID,
// 				})

// 				continue
// 			}
// 		}
// 	}
// }

// Size returns the total size of transports in the room.
func (t *PeerManager) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.webrtcTransports) + len(t.serverTransports)
}

func (t *PeerManager) Close() <-chan struct{} {
	ch := make(chan struct{}, 1)

	t.mu.Lock()

	for clientID, serverTransport := range t.serverTransports {
		t.log.Info("Closing server transport", logger.Ctx{
			"client_id": serverTransport.ClientID(),
		})

		serverTransport.Close()

		delete(t.serverTransports, clientID)
	}

	t.mu.Unlock()

	go func() {
		t.wg.Wait()
		t.pubsub.Close()

		close(ch)
	}()

	return ch
}
