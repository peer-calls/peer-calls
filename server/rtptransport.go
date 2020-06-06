package server

import (
	"fmt"
	"net"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

var receiveMTU = 8192

var ErrNoData = fmt.Errorf("cannot handle empty buffer")
var ErrUnknownPacket = fmt.Errorf("unknown packet")

type RTPTransport struct {
	conn   net.Conn
	rtpCh  chan *rtp.Packet
	rtcpCh chan rtcp.Packet // TODO change to []rtcp.Packet
	logger Logger
}

var _ Transport = &RTPTransport{}

func NewRTPTransport(loggerFactory LoggerFactory, conn net.Conn) *RTPTransport {

	t := RTPTransport{
		conn:   conn,
		rtpCh:  make(chan *rtp.Packet),
		rtcpCh: make(chan rtcp.Packet),
		logger: loggerFactory.GetLogger("rtctransport"),
	}

	go t.start()

	return &t
}

func (t *RTPTransport) start() {
	defer func() {
		close(t.rtcpCh)
		close(t.rtpCh)
	}()

	buf := make([]byte, receiveMTU)

	for {
		i, err := t.conn.Read(buf)
		if err != nil {
			t.logger.Printf("Error reading remote data: %s", err)
			return
		}

		err = t.handle(buf[:i])
		if err != nil {
			t.logger.Printf("Error handling remote data: %s", err)
		}
	}
}

func (t *RTPTransport) handle(buf []byte) error {
	if len(buf) == 0 {
		return ErrNoData
	}

	switch {
	case MatchRTP(buf):
		return t.handleRTP(buf)
	case MatchRTCP(buf):
		return t.handleRTCP(buf)
	default:
		return ErrUnknownPacket
	}
}

func (t *RTPTransport) handleRTP(buf []byte) error {
	pkt := &rtp.Packet{}
	err := pkt.Unmarshal(buf)
	if err != nil {
		return fmt.Errorf("Erorr unmarshalling RTP packet: %w", err)
	}
	t.rtpCh <- pkt
	return nil
}

func (t *RTPTransport) handleRTCP(buf []byte) error {
	pkts, err := rtcp.Unmarshal(buf)
	if err != nil {
		return fmt.Errorf("Error unmarshalling RTCP packet: %w", err)
	}
	// TODO we should probably keep RTCP packets together.
	for _, pkt := range pkts {
		t.rtcpCh <- pkt
	}
	return nil
}

func (t *RTPTransport) WriteRTCP(p []rtcp.Packet) error {
	b, err := rtcp.Marshal(p)
	if err != nil {
		return err
	}
	_, err = t.conn.Write(b)
	return err
}

func (t *RTPTransport) WriteRTP(p *rtp.Packet) (int, error) {
	b, err := p.Marshal()
	if err != nil {
		return 0, err
	}
	return t.conn.Write(b)
}

func (t *RTPTransport) RTPChannel() <-chan *rtp.Packet {
	return t.rtpCh
}

func (t *RTPTransport) RTCPChannel() <-chan rtcp.Packet {
	return t.rtcpCh
}

func (t *RTPTransport) Close() error {
	err := t.conn.Close()
	return err
}
