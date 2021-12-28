package servertransport

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
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
	params MetadataTransportParams

	localTracks  map[identifiers.TrackID]*trackLocalWithRTCPReader
	remoteTracks map[identifiers.TrackID]*trackRemoteWithRTCPReader
	mu           *sync.RWMutex

	// trackEventsCh chan transport.TrackEvent
	writeCh chan metadataEvent

	closeWriteLoop  chan struct{}
	writeLoopClosed chan struct{}
	readLoopClosed  chan struct{}

	remoteTracksChannel chan transport.TrackRemoteWithRTCPReader
}

type MetadataTransportParams struct {
	Log           logger.Logger
	Conn          io.ReadWriteCloser
	MediaStream   *MediaStream
	ClientID      identifiers.ClientID
	Interceptor   interceptor.Interceptor
	CodecRegistry *codecs.Registry
}

type trackLocalWithRTCPReader struct {
	trackLocal *trackLocal
	rtcpReader *rtcpReader
}

type trackRemoteWithRTCPReader struct {
	trackRemote *trackRemote
	rtcpReader  *rtcpReader
}

func NewMetadataTransport(params MetadataTransportParams) *MetadataTransport {
	params.Log = params.Log.WithNamespaceAppended("metadata_transport")

	t := &MetadataTransport{
		params: params,

		localTracks:  map[identifiers.TrackID]*trackLocalWithRTCPReader{},
		remoteTracks: map[identifiers.TrackID]*trackRemoteWithRTCPReader{},
		mu:           &sync.RWMutex{},

		// trackEventsCh: make(chan transport.TrackEvent),
		writeCh: make(chan metadataEvent),

		closeWriteLoop:  make(chan struct{}),
		writeLoopClosed: make(chan struct{}),
		readLoopClosed:  make(chan struct{}),

		remoteTracksChannel: make(chan transport.TrackRemoteWithRTCPReader),
	}

	t.params.Log.Trace("NewMetadataTransport", nil)

	go t.startReadLoop()
	go t.startWriteLoop()

	return t
}

func (t *MetadataTransport) startWriteLoop() {
	defer func() {
		close(t.writeLoopClosed)

		t.params.Log.Trace("Write closed", nil)
	}()

	write := func(event metadataEvent) error {
		t.params.Log.Trace("Write event", logger.Ctx{
			"metadata_event": event.Type,
		})

		b, err := json.Marshal(event)
		if err != nil {
			return errors.Trace(err)
		}

		_, err = t.params.Conn.Write(b)

		return errors.Trace(err)
	}

	for {
		select {
		case event := <-t.writeCh:
			if err := write(event); err != nil {
				t.params.Log.Error("Write", errors.Trace(err), nil)

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

		t.params.Log.Trace("Read closed", nil)
	}()

	buf := make([]byte, ReceiveMTU)

	for {
		i, err := t.params.Conn.Read(buf)
		if err != nil {
			t.params.Log.Error("Read", errors.Trace(err), nil)

			return
		}

		var event metadataEvent

		err = json.Unmarshal(buf[:i], &event)
		if err != nil {
			t.params.Log.Error("Unmarshal", err, nil)

			return
		}

		t.params.Log.Trace("Read event", logger.Ctx{
			"metadata_event": event.Type,
		})

		switch event.Type {
		case metadataEventTypeTrack:
			trackEv := event.TrackEvent
			track := trackEv.Track
			trackID := trackEv.Track.TrackID()

			switch trackEv.Type {
			case transport.TrackEventTypeAdd:
				// TODO simulcast rid

				t.mu.Lock()

				logCtx := logger.Ctx{
					"ssrc":      trackEv.SSRC,
					"track_id":  trackID,
					"client_id": t.params.ClientID,
				}

				subscribe := func() error {
					t.params.Log.Info("Sub", logCtx)

					err := t.sendTrackEvent(trackEvent{
						ClientID: t.params.ClientID,
						Track:    track,
						Type:     transport.TrackEventTypeSub,
						SSRC:     trackEv.SSRC,
					})

					return errors.Trace(err)
				}

				unsubscribe := func() error {
					t.params.Log.Info("Unsub", logCtx)

					err := t.sendTrackEvent(trackEvent{
						ClientID: t.params.ClientID,
						Track:    track,
						Type:     transport.TrackEventTypeUnsub,
						SSRC:     trackEv.SSRC,
					})

					return errors.Trace(err)
				}

				var (
					remoteTrack *trackRemote
					rtcpReader  *rtcpReader
				)

				if _, ok := t.remoteTracks[trackID]; ok {
					t.params.Log.Warn("Track already added", logCtx)
				} else {
					codec := track.Codec()
					interceptorParams, err := t.params.CodecRegistry.InterceptorParamsForCodec(codec)
					if err != nil {
						t.params.Log.Error("Unsupported codec", errors.Trace(err), nil)
					} else {
						remoteTrack = newTrackRemote(
							track,
							trackEv.SSRC,
							"", // TODO simulcast
							t.params.MediaStream.GetOrCreateBuffer(packetio.RTPBufferPacket, trackEv.SSRC),
							track.Codec(),
							t.params.Interceptor,
							interceptorParams,
							subscribe,
							unsubscribe,
						)

						rtcpReader = newRTCPReader(
							t.params.MediaStream.GetOrCreateBuffer(packetio.RTCPBufferPacket, trackEv.SSRC),
							t.params.Interceptor,
						)

						t.remoteTracks[trackID] = &trackRemoteWithRTCPReader{
							trackRemote: remoteTrack,
							rtcpReader:  rtcpReader,
						}
					}
				}

				t.mu.Unlock()

				if remoteTrack != nil {
					trwr := transport.TrackRemoteWithRTCPReader{
						TrackRemote: remoteTrack,
						RTCPReader:  rtcpReader,
					}

					// TODO potential deadlock.
					t.remoteTracksChannel <- trwr
				}
			case transport.TrackEventTypeRemove:
				t.mu.Lock()

				trwr, ok := t.remoteTracks[trackID]
				if ok {
					remoteTrack := trwr.trackRemote
					t.params.MediaStream.RemoveBuffer(packetio.RTPBufferPacket, remoteTrack.SSRC())
					delete(t.remoteTracks, trackID)
					remoteTrack.Close()
				}

				t.mu.Unlock()
			case transport.TrackEventTypeSub:
				t.mu.Lock()

				localTrack, ok := t.localTracks[trackID]

				t.mu.Unlock()

				if !ok {
					break
				}

				localTrack.trackLocal.subscribe()
			case transport.TrackEventTypeUnsub:
				t.mu.Lock()

				localTrack, ok := t.localTracks[trackID]

				t.mu.Unlock()

				if !ok {
					break
				}

				localTrack.trackLocal.unsubscribe()
			}
		default:
		}
	}
}

func (t *MetadataTransport) RemoteTracksChannel() <-chan transport.TrackRemoteWithRTCPReader {
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
		localTracks[i] = transport.NewTrackWithMID(localTrack.trackLocal.Track(), "")
	}

	return localTracks
}

func (t *MetadataTransport) AddTrack(track transport.Track) (transport.TrackLocal, transport.RTCPReader, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ssrc := webrtc.SSRC(RandUint32())
	codec := track.Codec()

	interceptorParams, err := t.params.CodecRegistry.InterceptorParamsForCodec(codec)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}

	// TODO I'm not sure if this is enough for simulcast.
	localTrack := newTrackLocal(
		track, t.params.MediaStream.Writer(),
		ssrc,
		track.Codec(),
		t.params.Interceptor,
		interceptorParams,
	)

	rtcpBuffer := t.params.MediaStream.GetOrCreateBuffer(packetio.RTCPBufferPacket, ssrc)
	sender := newRTCPReader(rtcpBuffer, t.params.Interceptor)

	t.localTracks[track.TrackID()] = &trackLocalWithRTCPReader{
		trackLocal: localTrack,
		rtcpReader: sender,
	}

	event := trackEvent{
		ClientID: t.params.ClientID,
		SSRC:     ssrc,
		Track:    track.SimpleTrack(),
		Type:     transport.TrackEventTypeAdd,
	}

	err = t.sendTrackEvent(event)

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

func (t *MetadataTransport) RemoveTrack(trackID identifiers.TrackID) error {
	t.mu.Lock()

	ltwr, ok := t.localTracks[trackID]
	delete(t.localTracks, trackID)

	t.mu.Unlock()

	if !ok {
		return errors.Errorf("remove track: not found: %s", trackID)
	}

	// Ensure writing stops and interceptors are released.
	ltwr.rtcpReader.Close()
	ltwr.trackLocal.Close()
	ssrc := ltwr.trackLocal.ssrc()

	// Ensure the RTCP buffer is closed. This will close the sender.
	t.params.MediaStream.RemoveBuffer(packetio.RTCPBufferPacket, ssrc)

	event := trackEvent{
		Track:    ltwr.trackLocal.Track().SimpleTrack(),
		SSRC:     ssrc,
		Type:     transport.TrackEventTypeRemove,
		ClientID: t.params.ClientID,
	}

	// TODO RemoveTrack should not be a slow operation.

	err := t.sendTrackEvent(event)

	return errors.Annotate(err, "send remove track event")
}

func (t *MetadataTransport) Close() error {
	err := t.params.Conn.Close()

	select {
	case t.closeWriteLoop <- struct{}{}:
		<-t.writeLoopClosed
	case <-t.writeLoopClosed:
	}

	<-t.readLoopClosed

	return errors.Trace(err)
}
