package servertransport

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/sfu"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

type MetadataTransport struct {
	clientID      string
	conn          io.ReadWriteCloser
	log           logger.Logger
	trackEventsCh chan transport.TrackEvent
	localTracks   map[uint32]transport.TrackInfo
	remoteTracks  map[uint32]transport.TrackInfo
	mu            *sync.Mutex
}

var _ transport.MetadataTransport = &MetadataTransport{}

func NewMetadataTransport(log logger.Logger, conn io.ReadWriteCloser, clientID string) *MetadataTransport {
	log = log.WithNamespaceAppended("server_metadata_transport")

	transport := &MetadataTransport{
		clientID:      clientID,
		log:           log,
		conn:          conn,
		localTracks:   map[uint32]transport.TrackInfo{},
		remoteTracks:  map[uint32]transport.TrackInfo{},
		trackEventsCh: make(chan transport.TrackEvent),
		mu:            &sync.Mutex{},
	}

	go transport.start()

	return transport
}

func (t *MetadataTransport) start() {
	defer close(t.trackEventsCh)

	buf := make([]byte, ReceiveMTU)

	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.log.Error("Read remote data", errors.Trace(err), nil)

			return
		}

		// hack because JSON does not know how to unmarshal to Track interface
		var eventJSON struct {
			TrackInfo struct {
				Track sfu.UserTrack
				Kind  webrtc.RTPCodecType
				Mid   string
			}
			Type transport.TrackEventType
		}

		err = json.Unmarshal(buf[:i], &eventJSON)
		if err != nil {
			t.log.Error("Unmarshal remote data", err, nil)

			return
		}

		track := &ServerTrack{
			UserTrack: eventJSON.TrackInfo.Track,
			onSub: func() error {
				t.log.Info("Sub", logger.Ctx{
					"ssrc":      eventJSON.TrackInfo.Track.SSRC(),
					"client_id": t.clientID,
				})

				err = t.sendTrackEvent(transport.TrackEvent{
					TrackInfo: transport.TrackInfo{
						Track: eventJSON.TrackInfo.Track,
						Kind:  eventJSON.TrackInfo.Kind,
						Mid:   eventJSON.TrackInfo.Mid,
					},
					ClientID: t.clientID,
					Type:     transport.TrackEventTypeSub,
				})

				return errors.Trace(err)
			},
			onUnsub: func() error {
				t.log.Info("Unsub", logger.Ctx{
					"ssrc":      eventJSON.TrackInfo.Track.SSRC(),
					"client_id": t.clientID,
				})

				err = t.sendTrackEvent(transport.TrackEvent{
					TrackInfo: transport.TrackInfo{
						Track: eventJSON.TrackInfo.Track,
						Kind:  eventJSON.TrackInfo.Kind,
						Mid:   eventJSON.TrackInfo.Mid,
					},
					ClientID: t.clientID,
					Type:     transport.TrackEventTypeSub,
				})

				return errors.Trace(err)
			},
		}

		trackEvent := transport.TrackEvent{
			TrackInfo: transport.TrackInfo{
				Track: track,
				Kind:  eventJSON.TrackInfo.Kind,
				Mid:   eventJSON.TrackInfo.Mid,
			},
			Type:     eventJSON.Type,
			ClientID: t.clientID,
		}

		switch trackEvent.Type {
		case transport.TrackEventTypeAdd:
			t.mu.Lock()
			t.remoteTracks[trackEvent.TrackInfo.Track.SSRC()] = trackEvent.TrackInfo
			t.mu.Unlock()
		case transport.TrackEventTypeRemove:
			t.mu.Lock()
			delete(t.remoteTracks, trackEvent.TrackInfo.Track.SSRC())
			t.mu.Unlock()
		case transport.TrackEventTypeSub:
		case transport.TrackEventTypeUnsub:
		}

		t.log.Info(fmt.Sprintf("Got track event: %+v", trackEvent), nil)

		t.trackEventsCh <- trackEvent
	}
}

func (t *MetadataTransport) TrackEventsChannel() <-chan transport.TrackEvent {
	return t.trackEventsCh
}

func (t *MetadataTransport) LocalTracks() []transport.TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	localTracks := make([]transport.TrackInfo, 0, len(t.localTracks))

	for _, trackInfo := range t.localTracks {
		localTracks = append(localTracks, trackInfo)
	}

	return localTracks
}

func (t *MetadataTransport) RemoteTracks() []transport.TrackInfo {
	t.mu.Lock()
	defer t.mu.Unlock()

	remoteTracks := make([]transport.TrackInfo, 0, len(t.remoteTracks))

	for _, trackInfo := range t.remoteTracks {
		remoteTracks = append(remoteTracks, trackInfo)
	}

	return remoteTracks
}

func (t *MetadataTransport) AddTrack(track transport.Track) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	trackInfo := transport.TrackInfo{
		Track: track,
		Kind:  t.getCodecType(track.PayloadType()),
		Mid:   "",
	}

	t.localTracks[track.SSRC()] = trackInfo

	trackEvent := transport.TrackEvent{
		TrackInfo: trackInfo,
		Type:      transport.TrackEventTypeAdd,
		ClientID:  t.clientID,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *MetadataTransport) sendTrackEvent(trackEvent transport.TrackEvent) error {
	b, err := json.Marshal(trackEvent)
	if err != nil {
		return errors.Annotatef(err, "sendTrackEvent: marshal")
	}

	_, err = t.conn.Write(b)

	return errors.Annotatef(err, "sendTrackEvent: write")
}

func (t *MetadataTransport) getCodecType(payloadType uint8) webrtc.RTPCodecType {
	// TODO These values are dynamic and are only valid when they are set in
	// media engine _and_ when we initiate peer connections.
	if payloadType == webrtc.DefaultPayloadTypeVP8 {
		return webrtc.RTPCodecTypeVideo
	}

	return webrtc.RTPCodecTypeAudio
}

func (t *MetadataTransport) RemoveTrack(ssrc uint32) error {
	t.mu.Lock()

	trackInfo, ok := t.localTracks[ssrc]
	delete(t.localTracks, ssrc)

	t.mu.Unlock()

	if !ok {
		return errors.Errorf("remove track: not found: %d", ssrc)
	}

	trackEvent := transport.TrackEvent{
		TrackInfo: trackInfo,
		Type:      transport.TrackEventTypeRemove,
		ClientID:  t.clientID,
	}

	return t.sendTrackEvent(trackEvent)
}

func (t *MetadataTransport) Close() error {
	return t.conn.Close()
}
