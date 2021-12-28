package play

import (
	"net"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/codecs"
	"github.com/peer-calls/peer-calls/v4/server/transport"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

// RTPReader is very similar to servertransport.rtcpReader. Perhaps unify
// them.
type RTPReader struct {
	params               RTPReaderParams
	streamInfo           *interceptor.StreamInfo
	interceptorRTPReader interceptor.RTPReader
}

type RTPReaderParams struct {
	Conn              *net.UDPConn
	Interceptor       interceptor.Interceptor
	SSRC              webrtc.SSRC
	Codec             transport.Codec
	InterceptorParams codecs.InterceptorParams
	MTU               int
}

func NewRTPReader(params RTPReaderParams) *RTPReader {
	r := &RTPReader{
		params: params,
	}

	r.streamInfo = &interceptor.StreamInfo{
		ID:                  "",
		Attributes:          nil,
		SSRC:                uint32(params.SSRC),
		PayloadType:         uint8(params.InterceptorParams.PayloadType),
		RTPHeaderExtensions: params.InterceptorParams.RTPHeaderExtensions,
		MimeType:            params.Codec.MimeType,
		ClockRate:           params.Codec.ClockRate,
		Channels:            params.Codec.Channels,
		SDPFmtpLine:         params.Codec.SDPFmtpLine,
		RTCPFeedback:        params.InterceptorParams.RTCPFeedback,
	}

	r.interceptorRTPReader = params.Interceptor.BindRemoteStream(
		r.streamInfo, interceptor.RTPReaderFunc(r.read),
	)

	return r
}

func (r *RTPReader) ReadRTP() (*rtp.Packet, interceptor.Attributes, error) {
	b := make([]byte, r.params.MTU)

	i, a, err := r.interceptorRTPReader.Read(b, interceptor.Attributes{})
	if err != nil {
		return nil, nil, errors.Annotatef(err, "read RTP")
	}

	var packet rtp.Packet

	err = packet.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTP")
	}

	return &packet, a, nil
}

func (r *RTPReader) read(in []byte, a interceptor.Attributes) (int, interceptor.Attributes, error) {
	i, err := r.params.Conn.Read(in)

	return i, a, errors.Trace(err)
}

func (r *RTPReader) Close() {
	r.params.Interceptor.UnbindRemoteStream(r.streamInfo)
	r.params.Conn.Close()
}
