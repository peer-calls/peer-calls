package sfu

// import (
// 	"github.com/juju/errors"
// 	"github.com/peer-calls/peer-calls/v4/server/transport"
// 	"github.com/pion/interceptor"
// 	"github.com/pion/rtcp"
// 	"github.com/pion/rtp"
// )

// type TrackWriter struct {
// 	track      transport.Track
// 	rtpWriter  RTPWriter
// 	rtcpWriter RTCPWriter
// 	rtcpReader RTCPReader
// }

// type RTPWriter interface {
// 	WriteRTP(*rtp.Packet) (int, error)
// }

// type RTCPReader interface {
// 	ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error)
// }

// type RTCPWriter interface {
// 	WriteRTCP([]rtcp.Packet) error
// }

// func NewTrackWriter(
// 	track transport.Track,
// 	trackLocal transport.TrackLocal,
// ) *TrackWriter {
// 	return &TrackWriter{
// 		track:      track,
// 		rtcpReader: trackLocal,
// 		rtcpWriter: trackLocal,
// 		rtpWriter:  trackLocal,
// 	}
// }

// var _ transport.TrackLocal = &TrackWriter{}

// func (t *TrackWriter) Track() transport.Track {
// 	return t.track
// }

// func (t *TrackWriter) WriteRTP(packet *rtp.Packet) (int, error) {
// 	n, err := t.rtpWriter.WriteRTP(packet)
// 	return n, errors.Trace(err)
// }

// func (t *TrackWriter) WriteRTCP(packets []rtcp.Packet) error {
// 	err := t.rtcpWriter.WriteRTCP(packets)
// 	return errors.Trace(err)
// }

// func (t *TrackWriter) ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error) {
// 	p, attributes, err := t.rtcpReader.ReadRTCP()
// 	return p, attributes, errors.Trace(err)
// }
