package play

import (
	"net"

	"github.com/juju/errors"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
)

// RTCPReader is very similar to servertransport.rtcpReader. Perhaps unify
// them.
type RTCPReader struct {
	params                RTCPReaderParams
	interceptorRTCPReader interceptor.RTCPReader
}

type RTCPReaderParams struct {
	Conn        *net.UDPConn
	Interceptor interceptor.Interceptor
	MTU         int
}

func NewRTCPReader(params RTCPReaderParams) *RTCPReader {
	r := &RTCPReader{
		params: params,
	}

	r.interceptorRTCPReader = params.Interceptor.BindRTCPReader(interceptor.RTCPReaderFunc(r.read))

	return r
}

func (r *RTCPReader) ReadRTCP() ([]rtcp.Packet, interceptor.Attributes, error) {
	b := make([]byte, r.params.MTU)

	i, a, err := r.interceptorRTCPReader.Read(b, interceptor.Attributes{})
	if err != nil {
		return nil, nil, errors.Annotatef(err, "reading RTCP")
	}

	packets, err := rtcp.Unmarshal(b[:i])
	if err != nil {
		return nil, nil, errors.Annotatef(err, "unmarshal RTCP")
	}

	return packets, a, nil
}

func (r *RTCPReader) read(in []byte, a interceptor.Attributes) (int, interceptor.Attributes, error) {
	i, err := r.params.Conn.Read(in)

	return i, a, errors.Trace(err)
}

func (r *RTCPReader) Close() {
	// TODO no way to unbind RTCPReader.
	r.params.Conn.Close()
}
