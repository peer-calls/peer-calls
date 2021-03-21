package server

//// trackBinding is a single bind for a Track
//// Bind can be called multiple times, this stores the
//// result for a single bind call so that it can be used when writing
//type trackBinding struct {
//	id          string
//	ssrc        webrtc.SSRC
//	payloadType webrtc.PayloadType
//	writeStream webrtc.TrackLocalWriter
//}

//// WebRTCTrackLocal  is a TrackLocal that has a pre-set codec and accepts RTP Packets.
//// If you wish to send a media.Sample use TrackLocalStaticSample
//type WebRTCTrackLocal struct {
//	mu           sync.RWMutex
//	bindings     []trackBinding
//	codec        webrtc.RTPCodecCapability
//	kind         webrtc.RTPCodecType
//	id, streamID string
//}

//var _ webrtc.TrackLocal = &WebRTCTrackLocal{}

//// NewWebRTCTrackLocal returns a WebRTCTrackLocal.
//func NewWebRTCTrackLocal(c webrtc.RTPCodecCapability, id, streamID string) (*WebRTCTrackLocal, error) {
//	var kind webrtc.RTPCodecType

//	switch {
//	case strings.HasPrefix(c.MimeType, "audio/"):
//		kind = webrtc.RTPCodecTypeAudio
//	case strings.HasPrefix(c.MimeType, "video/"):
//		kind = webrtc.RTPCodecTypeVideo
//	default:
//	}

//	return &WebRTCTrackLocal{
//		codec:    c,
//		bindings: []trackBinding{},
//		kind:     kind,
//		id:       id,
//		streamID: streamID,
//	}, nil
//}

//// Bind is called by the PeerConnection after negotiation is complete
//// This asserts that the code requested is supported by the remote peer.
//// If so it setups all the state (SSRC and PayloadType) to have a call
//func (s *WebRTCTrackLocal) Bind(t webrtc.TrackLocalContext) (webrtc.RTPCodecParameters, error) {
//	s.mu.Lock()
//	defer s.mu.Unlock()

//	/// FIXME only bind once because we'll be creating a separate track per
//	// peer connection.
//	//
//	// TODO perhaps that's an overkill and should be simplified.

//	parameters := webrtc.RTPCodecParameters{RTPCodecCapability: s.codec}
//	if codec, err := codecParametersFuzzySearch(parameters, t.CodecParameters()); err == nil {
//		s.bindings = append(s.bindings, trackBinding{
//			ssrc:        t.SSRC(),
//			payloadType: codec.PayloadType,
//			writeStream: t.WriteStream(),
//			id:          t.ID(),
//		})
//		return codec, nil
//	}

//	return webrtc.RTPCodecParameters{}, errors.Trace(webrtc.ErrUnsupportedCodec)
//}

//// Unbind implements the teardown logic when the track is no longer needed. This happens
//// because a track has been stopped.
//func (s *WebRTCTrackLocal) Unbind(t webrtc.TrackLocalContext) error {
//	s.mu.Lock()
//	defer s.mu.Unlock()

//	for i := range s.bindings {
//		if s.bindings[i].id == t.ID() {
//			s.bindings[i] = s.bindings[len(s.bindings)-1]
//			s.bindings = s.bindings[:len(s.bindings)-1]
//			return nil
//		}
//	}

//	return errors.Trace(webrtc.ErrUnbindFailed)
//}

//// ID is the unique identifier for this Track. This should be unique for the
//// stream, but doesn't have to globally unique. A common example would be 'audio' or 'video'
//// and StreamID would be 'desktop' or 'webcam'
//func (s *WebRTCTrackLocal) ID() string { return s.id }

//// StreamID is the group this track belongs too. This must be unique
//func (s *WebRTCTrackLocal) StreamID() string { return s.streamID }

//// Kind controls if this TrackLocal is audio or video
//func (s *WebRTCTrackLocal) Kind() webrtc.RTPCodecType {
//	return s.kind
//}

//// Codec gets the Codec of the track
//func (s *WebRTCTrackLocal) Codec() webrtc.RTPCodecCapability {
//	return s.codec
//}

//// WriteRTP writes a RTP Packet to the WebRTCTrackLocal
//// If one PeerConnection fails the packets will still be sent to
//// all PeerConnections. The error message will contain the ID of the failed
//// PeerConnections so you can remove them
//func (s *WebRTCTrackLocal) WriteRTP(p *rtp.Packet) error {
//	s.mu.RLock()
//	defer s.mu.RUnlock()

//	var errs multierr.MultiErr
//	outboundPacket := *p

//	for _, b := range s.bindings {
//		outboundPacket.Header.SSRC = uint32(b.ssrc)
//		outboundPacket.Header.PayloadType = uint8(b.payloadType)
//		if _, err := b.writeStream.WriteRTP(&outboundPacket.Header, outboundPacket.Payload); err != nil {
//			errs.Add(errors.Trace(err))
//		}
//	}

//	return errors.Trace(errs.Err())
//}

//// Write writes a RTP Packet as a buffer to the WebRTCTrackLocal
//// If one PeerConnection fails the packets will still be sent to
//// all PeerConnections. The error message will contain the ID of the failed
//// PeerConnections so you can remove them
//func (s *WebRTCTrackLocal) Write(b []byte) (n int, err error) {
//	packet := &rtp.Packet{}
//	if err = packet.Unmarshal(b); err != nil {
//		return 0, err
//	}

//	return len(b), s.WriteRTP(packet)
//}

//// Do a fuzzy find for a codec in the list of codecs
//// Used for lookup up a codec in an existing list to find a match
//func codecParametersFuzzySearch(
//	needle webrtc.RTPCodecParameters,
//	haystack []webrtc.RTPCodecParameters,
//) (webrtc.RTPCodecParameters, error) {
//	// First attempt to match on MimeType + SDPFmtpLine
//	for _, c := range haystack {
//		if strings.EqualFold(c.RTPCodecCapability.MimeType, needle.RTPCodecCapability.MimeType) &&
//			c.RTPCodecCapability.SDPFmtpLine == needle.RTPCodecCapability.SDPFmtpLine {
//			return c, nil
//		}
//	}

//	// Fallback to just MimeType
//	for _, c := range haystack {
//		if strings.EqualFold(c.RTPCodecCapability.MimeType, needle.RTPCodecCapability.MimeType) {
//			return c, nil
//		}
//	}

//	return webrtc.RTPCodecParameters{}, webrtc.ErrCodecNotFound
//}
