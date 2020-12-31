package sfu

type SubParams struct {
	// Room to which to subscribe to.
	Room        string
	PubClientID string
	SSRC        uint32
	SubClientID string
}
