package servertransport

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/webrtc/v3"
)

type MetadataTransport struct {
	clientID string
	conn     io.ReadWriteCloser
	log      logger.Logger

	localTracks  map[uint32]transport.TrackInfo
	remoteTracks map[uint32]transport.TrackInfo
	mu           *sync.RWMutex

	trackEventsCh chan transport.TrackEvent
	writeCh       chan metadataEvent

	closeWriteLoop  chan struct{}
	writeLoopClosed chan struct{}
	readLoopClosed  chan struct{}
}

var _ transport.MetadataTransport = &MetadataTransport{}

func NewMetadataTransport(log logger.Logger, conn io.ReadWriteCloser, clientID string) *MetadataTransport {
	log = log.WithNamespaceAppended("metadata_transport")

	t := &MetadataTransport{
		clientID:     clientID,
		log:          log,
		conn:         conn,
		localTracks:  map[uint32]transport.TrackInfo{},
		remoteTracks: map[uint32]transport.TrackInfo{},
		mu:           &sync.RWMutex{},

		trackEventsCh: make(chan transport.TrackEvent),
		writeCh:       make(chan metadataEvent),

		closeWriteLoop:  make(chan struct{}),
		writeLoopClosed: make(chan struct{}),
		readLoopClosed:  make(chan struct{}),
	}

	log.Trace("NewMetadataTransport", nil)

	go t.startReadLoop()
	go t.startWriteLoop()

	return t
}

func (t *MetadataTransport) newServerTrack(trackInfo trackInfoJSON) *ServerTrack {
	return &ServerTrack{
		UserTrack: trackInfo.Track,
		onSub: func() error {
			t.log.Info("Sub", logger.Ctx{
				"ssrc":      trackInfo.Track.SSRC(),
				"client_id": t.clientID,
			})

			err := t.sendTrackEvent(transport.TrackEvent{
				TrackInfo: transport.TrackInfo{
					Track: trackInfo.Track,
					Kind:  trackInfo.Kind,
					Mid:   trackInfo.Mid,
				},
				ClientID: t.clientID,
				Type:     transport.TrackEventTypeSub,
			})

			return errors.Trace(err)
		},
		onUnsub: func() error {
			t.log.Info("Unsub", logger.Ctx{
				"ssrc":      trackInfo.Track.SSRC(),
				"client_id": t.clientID,
			})

			err := t.sendTrackEvent(transport.TrackEvent{
				TrackInfo: transport.TrackInfo{
					Track: trackInfo.Track,
					Kind:  trackInfo.Kind,
					Mid:   trackInfo.Mid,
				},
				ClientID: t.clientID,
				Type:     transport.TrackEventTypeSub,
			})

			return errors.Trace(err)
		},
	}
}

func (t *MetadataTransport) startWriteLoop() {
	defer func() {
		close(t.writeLoopClosed)

		t.log.Trace("Write closed", nil)
	}()

	write := func(event metadataEvent) error {
		t.log.Trace("Write event", logger.Ctx{
			"metadata_event": event.Type,
		})

		b, err := json.Marshal(event)
		if err != nil {
			return errors.Trace(err)
		}

		_, err = t.conn.Write(b)

		return errors.Trace(err)
	}

	for {
		select {
		case event := <-t.writeCh:
			if err := write(event); err != nil {
				t.log.Error("Write", errors.Trace(err), nil)

				continue
			}
		case <-t.closeWriteLoop:
			return
		}
	}
}

func (t *MetadataTransport) startReadLoop() {
	defer func() {
		close(t.trackEventsCh)
		close(t.readLoopClosed)

		t.log.Trace("Read closed", nil)
	}()

	buf := make([]byte, ReceiveMTU)

	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.log.Error("Read", errors.Trace(err), nil)

			return
		}

		var event metadataEvent

		err = json.Unmarshal(buf[:i], &event)
		if err != nil {
			t.log.Error("Unmarshal", err, nil)

			return
		}

		t.log.Trace("Read event", logger.Ctx{
			"metadata_event": event.Type,
		})

		switch event.Type {
		case metadataEventTypeTrack:
			trackEvent := event.Track.trackEvent(t.clientID)
			trackEvent.TrackInfo.Track = t.newServerTrack(event.Track.TrackInfo)

			skipEvent := false

			switch trackEvent.Type {
			case transport.TrackEventTypeAdd:
				ssrc := trackEvent.TrackInfo.Track.SSRC()
				t.mu.Lock()
				// Skip event in case of a refresh event, and track information has
				// already been received.
				_, skipEvent = t.remoteTracks[ssrc]
				t.remoteTracks[ssrc] = trackEvent.TrackInfo
				t.mu.Unlock()
			case transport.TrackEventTypeRemove:
				t.mu.Lock()
				delete(t.remoteTracks, trackEvent.TrackInfo.Track.SSRC())
				t.mu.Unlock()
			case transport.TrackEventTypeSub:
			case transport.TrackEventTypeUnsub:
			}

			if !skipEvent {
				select {
				case t.trackEventsCh <- trackEvent:
				case <-t.writeLoopClosed:
				}
			}
		}
	}
}

func (t *MetadataTransport) TrackEventsChannel() <-chan transport.TrackEvent {
	return t.trackEventsCh
}

func (t *MetadataTransport) LocalTracks() []transport.TrackInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	localTracks := make([]transport.TrackInfo, 0, len(t.localTracks))

	for _, trackInfo := range t.localTracks {
		localTracks = append(localTracks, trackInfo)
	}

	return localTracks
}

func (t *MetadataTransport) RemoteTracks() []transport.TrackInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

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

	err := t.sendTrackEvent(trackEvent)

	return errors.Trace(err)
}

func (t *MetadataTransport) sendTrackEvent(trackEvent transport.TrackEvent) error {
	json := newTrackEventJSON(trackEvent)

	err := t.sendMetadataEvent(metadataEvent{
		Type:  metadataEventTypeTrack,
		Track: &json,
	})

	return errors.Annotatef(err, "sendTrackEvent: write")
}

func (t *MetadataTransport) sendMetadataEvent(event metadataEvent) error {
	select {
	case t.writeCh <- event:
		return nil
	case <-t.writeLoopClosed:
		return errors.Annotatef(io.ErrClosedPipe, "sendMetadataEvent: write")
	}
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

	// TODO RemoveTrack should not be a slow operation.

	return t.sendTrackEvent(trackEvent)
}

func (t *MetadataTransport) Close() error {
	err := t.conn.Close()

	select {
	case t.closeWriteLoop <- struct{}{}:
		<-t.writeLoopClosed
	case <-t.writeLoopClosed:
	}

	<-t.readLoopClosed

	return errors.Trace(err)
}
