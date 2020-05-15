package server

import (
	"sync"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

// JitterBuffer contains ring buffers for RTP packets per track SSRC
type JitterBuffer struct {
	mu      sync.Mutex
	buffers map[uint32]*Buffer
}

func NewJitterBuffer() *JitterBuffer {
	return &JitterBuffer{
		buffers: make(map[uint32]*Buffer),
	}
}

// PushRTP pushes a RTP packet to buffer and returns a Nack RTCP packet if
// the buffer determines that there is a missing packet.
func (j *JitterBuffer) PushRTP(p *rtp.Packet) rtcp.Packet {
	j.mu.Lock()
	defer j.mu.Unlock()

	buffer, ok := j.buffers[p.SSRC]
	if !ok {
		buffer = NewBuffer()
		j.buffers[p.SSRC] = buffer
	}

	return buffer.Push(p)
}

// GetPacket retreives a packet from the RTP buffer
func (j *JitterBuffer) GetPacket(ssrc uint32, sn uint16) *rtp.Packet {
	j.mu.Lock()
	defer j.mu.Unlock()

	buffer, ok := j.buffers[ssrc]
	if !ok {
		return nil
	}
	return buffer.GetPacket(sn)
}

func (j *JitterBuffer) RemoveBuffer(ssrc uint32) {
	j.mu.Lock()
	defer j.mu.Unlock()

	delete(j.buffers, ssrc)
}
