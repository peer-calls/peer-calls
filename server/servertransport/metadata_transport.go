package servertransport

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/transport"
	"github.com/pion/randutil"
	"github.com/pion/transport/packetio"
	"github.com/pion/webrtc/v3"
)

// Use global random generator to properly seed by crypto grade random.
var globalMathRandomGenerator = randutil.NewMathRandomGenerator() // nolint:gochecknoglobals

// RandUint32 generates a mathmatical random uint32.
func RandUint32() uint32 {
	return globalMathRandomGenerator.Uint32()
}

type MetadataTransport struct {
	clientID string
	conn     io.ReadWriteCloser
	log      logger.Logger

	localTracks  map[transport.TrackID]*trackLocal
	remoteTracks map[transport.TrackID]*trackRemote
	mu           *sync.RWMutex

	// trackEventsCh chan transport.TrackEvent
	writeCh chan metadataEvent

	mediaTransport *MediaTransport

	closeWriteLoop  chan struct{}
	writeLoopClosed chan struct{}
	readLoopClosed  chan struct{}

	remoteTracksChannel chan transport.TrackRemote
}

type BufferFactory func(ssrc uint32) *packetio.Buffer

func NewMetadataTransport(
	log logger.Logger,
	conn io.ReadWriteCloser,
	mediaTransport *MediaTransport,
	clientID string,
) *MetadataTransport {
	log = log.WithNamespaceAppended("metadata_transport")

	t := &MetadataTransport{
		clientID:     clientID,
		log:          log,
		conn:         conn,
		localTracks:  map[transport.TrackID]*trackLocal{},
		remoteTracks: map[transport.TrackID]*trackRemote{},
		mu:           &sync.RWMutex{},

		// trackEventsCh: make(chan transport.TrackEvent),
		writeCh: make(chan metadataEvent),

		closeWriteLoop:  make(chan struct{}),
		writeLoopClosed: make(chan struct{}),
		readLoopClosed:  make(chan struct{}),

		remoteTracksChannel: make(chan transport.TrackRemote),

		mediaTransport: mediaTransport,
	}

	log.Trace("NewMetadataTransport", nil)

	go t.startReadLoop()
	go t.startWriteLoop()

	return t
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
		// close(t.trackEventsCh)
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
			trackEvent := event.TrackEvent
			track := trackEvent.Track
			trackID := trackEvent.Track.UniqueID()

			switch trackEvent.Type {
			case transport.TrackEventTypeAdd:
				// TODO simulcast rid

				t.mu.Lock()

				var remoteTrack *trackRemote

				_, ok := t.remoteTracks[trackID]
				if !ok {
					remoteTrack = newTrackRemote(
						track,
						trackEvent.SSRC,
						"",
						t.mediaTransport.getOrCreateRTPBuffer(trackEvent.SSRC),
					)

					t.remoteTracks[trackID] = remoteTrack
				}

				t.mu.Unlock()

				if remoteTrack != nil {
					// TODO potential deadlock.
					t.remoteTracksChannel <- remoteTrack
				}
			case transport.TrackEventTypeRemove:
				t.mu.Lock()

				remoteTrack, ok := t.remoteTracks[trackID]
				if ok {
					t.mediaTransport.removeRTPBuffer(remoteTrack.SSRC())
					delete(t.remoteTracks, trackID)
				}

				t.mu.Unlock()
			case transport.TrackEventTypeSub:
				t.mu.Lock()

				localTrack, ok := t.localTracks[trackID]

				t.mu.Unlock()

				if !ok {
					break
				}

				localTrack.subscribe()
			case transport.TrackEventTypeUnsub:
				t.mu.Lock()

				localTrack, ok := t.localTracks[trackID]

				t.mu.Unlock()

				if !ok {
					break
				}

				localTrack.unsubscribe()
			}
		}
	}
}

func (t *MetadataTransport) RemoteTracksChannel() <-chan transport.TrackRemote {
	return t.remoteTracksChannel
}

// func (t *MetadataTransport) TrackEventsChannel() <-chan transport.TrackEvent {
// 	return t.trackEventsCh
// }

func (t *MetadataTransport) LocalTracks() []transport.TrackWithMID {
	t.mu.RLock()
	defer t.mu.RUnlock()

	localTracks := make([]transport.TrackWithMID, len(t.localTracks))

	i := -1
	for _, localTrack := range t.localTracks {
		i++
		localTracks[i] = transport.NewTrackWithMID(localTrack.Track(), "")
	}

	return localTracks
}

// func (t *MetadataTransport) RemoteTracks() []transport.Track {
// 	t.mu.RLock()
// 	defer t.mu.RUnlock()

// 	remoteTracks := make([]transport.Track, len(t.remoteTracks))

// 	i := -1
// 	for _, track := range t.remoteTracks {
// 		i++
// 		remoteTracks[i] = track
// 	}

// 	return remoteTracks
// }

func (t *MetadataTransport) AddTrack(track transport.Track) (transport.TrackLocal, transport.Sender, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ssrc := webrtc.SSRC(RandUint32())

	// trackInfo := transport.NewTrackWithMID(track, "")

	localTrack := newTrackLocal(track, t.mediaTransport.conn, ssrc)
	sender := newSender(t.mediaTransport.getOrCreateRTCPBuffer(ssrc))

	t.localTracks[track.UniqueID()] = localTrack

	event := trackEvent{
		ClientID: t.clientID,
		SSRC:     ssrc,
		Track:    track.SimpleTrack(),
		Type:     transport.TrackEventTypeAdd,
	}

	err := t.sendTrackEvent(event)

	return localTrack, sender, errors.Trace(err)
}

func (t *MetadataTransport) sendTrackEvent(event trackEvent) error {
	err := t.sendMetadataEvent(metadataEvent{
		Type:       metadataEventTypeTrack,
		TrackEvent: event,
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

func (t *MetadataTransport) RemoveTrack(trackID transport.TrackID) error {
	t.mu.Lock()

	localTrack, ok := t.localTracks[trackID]
	delete(t.localTracks, trackID)

	t.mu.Unlock()

	if !ok {
		return errors.Errorf("remove track: not found: %s", trackID)
	}

	// Ensure the RTCP buffer is closed. This will close the sender.
	t.mediaTransport.removeRTCPBuffer(localTrack.ssrc)

	event := trackEvent{
		Track:    localTrack.Track().SimpleTrack(),
		SSRC:     localTrack.ssrc,
		Type:     transport.TrackEventTypeRemove,
		ClientID: t.clientID,
	}

	// TODO RemoveTrack should not be a slow operation.

	err := t.sendTrackEvent(event)

	return errors.Annotate(err, "send remove track event")
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
