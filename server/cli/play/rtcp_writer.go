package play

import (
	"net"

	"github.com/juju/errors"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
)

type RTCPWriter struct {
	params                RTCPWriterParams
	interceptorRTCPWriter interceptor.RTCPWriter
}

type RTCPWriterParams struct {
	Conn        *net.UDPConn
	Interceptor interceptor.Interceptor
	MTU         int
}

func NewRTCPWriter(params RTCPWriterParams) *RTCPWriter {
	r := &RTCPWriter{
		params: params,
	}

	r.interceptorRTCPWriter = params.Interceptor.BindRTCPWriter(interceptor.RTCPWriterFunc(r.writeRTCP))

	return r
}

func (r *RTCPWriter) WriteRTCP(p []rtcp.Packet) error {
	_, err := r.interceptorRTCPWriter.Write(p, make(interceptor.Attributes))

	return errors.Trace(err)
}

func (r *RTCPWriter) writeRTCP(p []rtcp.Packet, _ interceptor.Attributes) (int, error) {
	b, err := rtcp.Marshal(p)
	if err != nil {
		return 0, errors.Annotatef(err, "marshal RTCP")
	}

	i, err := r.params.Conn.Write(b)

	return i, errors.Annotatef(err, "write RTCP")
}

func (r *RTCPWriter) Close() {
	// TODO no way to unbind RTCPWriter.
	r.params.Conn.Close()
}
