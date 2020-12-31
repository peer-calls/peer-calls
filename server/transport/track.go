package transport

type Track interface {
	PayloadType() uint8
	SSRC() uint32
	ID() string
	Label() string
}
