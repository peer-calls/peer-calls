package server

import (
	"testing"

	"github.com/pion/rtp"
	"github.com/stretchr/testify/assert"
)

func TestJitterBuffer(t *testing.T) {
	assert := assert.New(t)

	j := NewJitterBuffer()

	p1 := rtp.Packet{}
	p1.SequenceNumber = 15
	p1.SSRC = 123
	j.PushRTP(&p1)
	p2 := rtp.Packet{}
	p2.SequenceNumber = 16
	p2.SSRC = p1.SSRC
	j.PushRTP(&p2)

	assert.Equal(&p1, j.GetPacket(p1.SSRC, p1.SequenceNumber))
	assert.Equal(&p2, j.GetPacket(p2.SSRC, p2.SequenceNumber))
	assert.Nil(j.GetPacket(p1.SSRC, 456))
	j.RemoveBuffer(p1.SSRC)
	assert.Nil(j.GetPacket(p1.SSRC, p1.SequenceNumber))
}
