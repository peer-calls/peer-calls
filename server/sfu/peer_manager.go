package sfu

import (
	"io"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/clock"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/multierr"
	"github.com/peer-calls/peer-calls/v4/server/pubsub"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

var ErrDuplicateTransport = errors.New("duplicate transport")

type PeerManager struct {
	log logger.Logger
	mu  sync.RWMutex
	wg  sync.WaitGroup

	jitterHandler JitterHandler

	// transports indexed by ClientID
	transports map[identifiers.ClientID]transport.Transport

	pliTimes map[identifiers.TrackID]time.Time

	room identifiers.RoomID

	// pubsub keeps track of published tracks and its subscribers.
	pubsub *pubsub.PubSub
}

func NewPeerManager(room identifiers.RoomID, log logger.Logger, jitterHandler JitterHandler) *PeerManager {
	return &PeerManager{
		log: log.WithNamespaceAppended("room_peers_manager"),

		jitterHandler: jitterHandler,

		transports: map[identifiers.ClientID]transport.Transport{},

		pliTimes: map[identifiers.TrackID]time.Time{},

		room: room,

		pubsub: pubsub.New(log, clock.New()),
	}
}

func (t *PeerManager) broadcast(clientID identifiers.ClientID, msg webrtc.DataChannelMessage) {
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

// Add adds a transport with ClientID. If there was already an existing
// Transport with the same ClientID, it will be closed and removed before a new
// one is added.
func (t *PeerManager) Add(tr transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	clientID := tr.ClientID()

	log := t.log.WithCtx(logger.Ctx{
		"client_id": clientID,
	})

	t.mu.Lock()

	pubTrackEventSub, err := t.add(tr)

	t.mu.Unlock()

	if err != nil {
		return nil, errors.Annotatef(err, "subscribe to events: %s", clientID)
	}

	pubTracks := t.pubsub.Tracks()

	pubTrackEventsCh := make(chan pubsub.PubTrackEvent)

	t.wg.Add(1)

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		defer close(pubTrackEventsCh)

		for _, pubTrack := range pubTracks {
			if pubTrack.ClientID != clientID {
				pubTrackEventsCh <- pubsub.PubTrackEvent{
					PubTrack: pubTrack,
					Type:     transport.TrackEventTypeAdd,
				}
			}
		}

		for event := range pubTrackEventSub {
			if event.PubTrack.ClientID != clientID {
				pubTrackEventsCh <- event
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		remoteTracksCh := tr.RemoteTracksChannel()
		doneCh := tr.Done()

		for {
			select {
			case remoteTrackWithReceiver := <-remoteTracksCh:
				remoteTrack := remoteTrackWithReceiver.TrackRemote
				rtcpReader := remoteTrackWithReceiver.RTCPReader
				trackID := remoteTrack.Track().TrackID()

				done := make(chan struct{})

				t.pubsub.Pub(clientID, pubsub.NewTrackReader(remoteTrack, func() {
					t.mu.Lock()

					close(done)

					t.pubsub.Unpub(clientID, trackID)

					t.mu.Unlock()
				}))

				t.wg.Add(1)

				go func() {
					defer t.wg.Done()

					for {
						// ReadRTCP ensures interceptors will do their work.
						packets, _, err := rtcpReader.ReadRTCP()
						if err != nil {
							if !multierr.Is(err, io.EOF) {
								log.Error("ReadRTCP from receiver", errors.Trace(err), nil)
							}

							return
						}

						prometheusRTCPPacketsReceived.Add(float64(len(packets)))
					}
				}()

				t.wg.Add(1)

				ticker := time.NewTicker(time.Second)

				go func() {
					defer func() {
						t.wg.Done()
						ticker.Stop()
					}()

					getBitrateEstimate := func() (float32, bool) {
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

						if err == nil {
							prometheusRTCPPacketsSent.Inc()
						}

						_ = err // FIXME handle error

					case <-done:
					}
				}()
			case <-doneCh:
				return
			}
		}
	}()

	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for msg := range tr.MessagesChannel() {
			t.broadcast(clientID, msg)
		}
	}()

	t.wg.Done()

	return pubTrackEventsCh, nil
}

// add removes and closes any existing transport with the same clientID and
// subscribes to events and adds the new transport. The caller must hold the
// lock.
func (t *PeerManager) add(tr transport.Transport) (<-chan pubsub.PubTrackEvent, error) {
	clientID := tr.ClientID()

	// Remove and close an existing transport so we don't have troubling cleaning
	// up.
	if existing, ok := t.transports[clientID]; ok {
		// TODO perhaps it would be wise to wait for all the previously spinned
		// goroutines created in the past call to this method to exit before we
		// remove the transport to prevent any stale tracks from being added.
		existing.Close()

		t.remove(clientID)
	}

	pubTrackEventSub, err := t.pubsub.SubscribeToEvents(clientID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Add the transport only if there was no error. This awkward check is here
	// because we're still under a lock.
	t.transports[clientID] = tr

	return pubTrackEventSub, nil
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

		feedBitrateEstimate := func(trackID identifiers.TrackID, bitrate float32) {
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
				// return errors.Errorf("too many PLI packets received, ignoring")
				return nil
			}

			// Important: set the correct SSRC before sending the packet to source.
			packet.MediaSSRC = uint32(props.SSRC)
			packet.SenderSSRC = uint32(props.SSRC)

			if err := transport.WriteRTCP([]rtcp.Packet{packet}); err != nil {
				return errors.Annotatef(err, "sending PLI back to source: %s", props.ClientID)
			}

			prometheusRTCPPacketsSent.Inc()

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
				prometheusRTCPPLIPacketsReceived.Inc()
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

			prometheusRTCPPacketsReceived.Add(float64(len(packets)))

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

// Remove removes the transport and unsubscribes it from track events. To
// qualify for removal, the registered transport must have the same reference,
// otherwise it will not be removed. This is to prevent a transport with the
// same ID from messing up with another transport. This means that Remove can
// be called after a transport was already replaced in Add to ease the cleanup
// logic.
func (t *PeerManager) Remove(tr transport.Transport) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	clientID := tr.ClientID()

	if existing, ok := t.transports[clientID]; !ok {
		return errors.Errorf("transport not found: %s", clientID)
	} else if existing != tr {
		// Most likely a transport was already removed after a reconnect.
		return errors.Errorf("transport found, but has changed: %s", clientID)
	}

	t.remove(clientID)

	return nil
}

// remove unsubscribes the transport from track events and removes any
// published published tracks. The transport should be closed by the time this
// method is called. The caller must hold the lock.
func (t *PeerManager) remove(clientID identifiers.ClientID) {
	t.log.Trace("Remove", logger.Ctx{
		"client_id": clientID,
	})

	if err := t.pubsub.UnsubscribeFromEvents(clientID); err != nil {
		t.log.Error("Unsubscribe from events", errors.Trace(err), logger.Ctx{
			"client_id": clientID,
		})
	}

	t.pubsub.Terminate(clientID)

	delete(t.transports, clientID)
}

// Size returns the total size of transports in the room.
func (t *PeerManager) Size() int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return len(t.transports)
}

func (t *PeerManager) Close() <-chan struct{} {
	t.log.Info("Close PeerManager", nil)

	ch := make(chan struct{})

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
		// TODO there is a race condition here but I was unable to reproduce it the
		// second time. This method will get called twice if we allow two clients
		// with the same id to join. It panics because it tries to close the closed
		// channel twice.
		t.pubsub.Close()

		close(ch)
	}()

	return ch
}
