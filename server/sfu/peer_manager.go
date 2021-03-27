package sfu

import (
	"io"
	"strings"
	"sync"
	"time"

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

	jitterHandler JitterHandler

	// transports indexed by ClientID
	transports map[string]transport.Transport

	pliTimes map[transport.TrackID]time.Time

	room string

	// // pubsub keeps track of published tracks and its subscribers.
	pubsub *pubsub.PubSub
}

func NewPeerManager(room string, log logger.Logger, jitterHandler JitterHandler) *PeerManager {
	return &PeerManager{
		log: log.WithNamespaceAppended("room_peers_manager"),

		jitterHandler: jitterHandler,

		transports: map[string]transport.Transport{},

		pliTimes: map[transport.TrackID]time.Time{},

		room: room,

		pubsub: pubsub.New(log),
	}
}

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

	for _, tr := range t.transports {
		broadcast(tr)
	}
}

func (t *PeerManager) Add(tr transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	log := t.log.WithCtx(logger.Ctx{
		"client_id": tr.ClientID(),
	})

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

		for remoteTrackWithReceiver := range tr.RemoteTracksChannel() {
			remoteTrack := remoteTrackWithReceiver.TrackRemote
			rtcpReader := remoteTrackWithReceiver.RTCPReader
			trackID := remoteTrack.Track().UniqueID()

			done := make(chan struct{})

			t.pubsub.Pub(tr.ClientID(), pubsub.NewTrackReader(remoteTrack, func() {
				t.mu.Lock()

				close(done)

				t.pubsub.Unpub(tr.ClientID(), trackID)

				t.mu.Unlock()
			}))

			t.wg.Add(1)

			go func() {
				defer t.wg.Done()

				for {
					// ReadRTCP ensures interceptors will do their work.
					_, _, err := rtcpReader.ReadRTCP()
					if err != nil {
						if !multierr.Is(err, io.EOF) {
							log.Error("ReadRTCP from receiver", errors.Trace(err), nil)
						}

						return
					}
				}
			}()

			t.wg.Add(1)

			ticker := time.NewTicker(time.Second)

			go func() {
				defer func() {
					t.wg.Done()
					ticker.Stop()
				}()

				getBitrateEstimate := func() (uint64, bool) {
					t.mu.Lock()
					defer t.mu.Unlock()

					estimator, ok := t.pubsub.BitrateEstimator(trackID)

					if !ok || estimator.Empty() {
						return 0, false
					}

					return estimator.Min(), true
				}

				select {
				case <-ticker.C:
					bitrate, ok := getBitrateEstimate()
					if !ok {
						break
					}

					ssrc := uint32(remoteTrack.SSRC())

					// FIXME simulcast?

					err := tr.WriteRTCP([]rtcp.Packet{
						&rtcp.ReceiverEstimatedMaximumBitrate{
							SenderSSRC: ssrc,
							Bitrate:    bitrate,
							SSRCs:      []uint32{ssrc},
						},
					})
					_ = err // FIXME handle error

				case <-done:
				}
			}()
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
	t.transports[tr.ClientID()] = tr
	t.mu.Unlock()

	return pubTrackEventsCh, nil
}

func (t *PeerManager) Sub(params SubParams) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	tr, ok := t.transports[params.SubClientID]
	if !ok {
		return errors.Errorf("transport not found: %s", params.PubClientID)
	}

	rtcpReader, err := t.pubsub.Sub(params.PubClientID, params.TrackID, tr)
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

		feedBitrateEstimate := func(trackID transport.TrackID, bitrate uint64) {
			t.mu.Lock()

			bitrateEstimator, ok := t.pubsub.BitrateEstimator(trackID)
			if ok {
				bitrateEstimator.Feed(params.SubClientID, bitrate)
			}

			t.mu.Unlock()
		}

		forwardPLI := func(packet *rtcp.PictureLossIndication) error {
			now := time.Now()

			t.mu.Lock()

			props, propsFound := t.pubsub.TrackPropsByTrackID(params.TrackID)
			transport, transportFound := t.transports[props.ClientID]
			lastPLITime := t.pliTimes[params.TrackID]

			// TODO perhaps a better solution for this would be an RTCP interceptor.
			pliTooSoon := now.Sub(lastPLITime) < time.Second
			if !pliTooSoon {
				t.pliTimes[params.TrackID] = now
			}

			t.mu.Unlock()

			if !propsFound {
				return errors.Annotatef(pubsub.ErrTrackNotFound, "got RTCP for track that was not found")
			}

			if !transportFound {
				return errors.Errorf("transport not found: %s", props.ClientID)
			}

			if pliTooSoon {
				// Congestion control.
				return errors.Errorf("too many PLI packets received, ignoring")
			}

			// Important: set the correct SSRC before sending the packet to source.
			packet.MediaSSRC = uint32(props.SSRC)
			packet.SenderSSRC = uint32(props.SSRC)

			if err := transport.WriteRTCP([]rtcp.Packet{packet}); err != nil {
				return errors.Annotatef(err, "sending PLI back to source: %s", props.ClientID)
			}

			// TODO remove this log.
			t.log.Info("Sent PLI back to source", logCtx)

			return nil
		}

		handlePacket := func(p rtcp.Packet) (err error) {
			// NOTE: REMB and NACK are now handled by pion/webrtc interceptors so we
			// don't have to explicitly handle them here.
			switch packet := p.(type) {
			// PLI cannot be handled by interceptors since it's implementation
			// specific. We need to find the source and send the PLI packet. We also
			// need to make sure to set the correct SSRC before the packet is
			// forwarded, since pion/webrtc/v3 no longer uses the same SSRCs between
			// different peer connections.
			case *rtcp.PictureLossIndication:
				err = errors.Trace(forwardPLI(packet))
			case *rtcp.ReceiverEstimatedMaximumBitrate:
				feedBitrateEstimate(params.TrackID, packet.Bitrate)
			default:
			}

			return errors.Trace(err)
		}

		for {
			packets, _, err := rtcpReader.ReadRTCP()
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

	tr, ok := t.transports[clientID]
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

	delete(t.transports, clientID)
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

	return len(t.transports)
}

func (t *PeerManager) Close() <-chan struct{} {
	ch := make(chan struct{}, 1)

	t.mu.Lock()

	// This is only needed for server transports.
	for clientID, transport := range t.transports {
		t.log.Info("Closing transport", logger.Ctx{
			"client_id": transport.ClientID(),
		})

		transport.Close()

		delete(t.transports, clientID)
	}

	t.mu.Unlock()

	go func() {
		t.wg.Wait()
		t.pubsub.Close()

		close(ch)
	}()

	return ch
}
