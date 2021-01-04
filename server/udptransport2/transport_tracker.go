package udptransport2

// transportTracker ensures that local and remote transports are both created
// before adding them to the rooms. It also ensures that the remote one is
// removed before closing the transport.
type transportTracker struct {
	transport *Transport

	lastLocalEvent  controlEventType
	lastRemoteEvent controlEventType
}

func (t *transportTracker) setTransport(transport *Transport) {
	t.transport = transport
}

func (t *transportTracker) setLocalEvent(event controlEventType) controlAction {
	t.lastLocalEvent = event

	return t.nextAction()
}

func (t *transportTracker) setRemoteEvent(event controlEventType) controlAction {
	t.lastRemoteEvent = event

	return t.nextAction()
}

func (t *transportTracker) nextAction() controlAction {
	if t.lastLocalEvent == t.lastRemoteEvent {
		switch t.lastLocalEvent {
		case controlEventTypeInit:
			return controlActionAdd
		case controlEventTypeDestroy:
			return controlActionRemove
		}
	}

	return controlActionWait
}

//                   node 1                 node 2
//     peer join  +-----------+             +-----------+
//    ----------> | create    |   init      |           |
//                | transport | ----------> |           |
//                |           |             |           |
//                |           |   init      | create    |   peer join
//                |           | <---------- | transport | <-----------
//                |           |             |           |
//                | add       |             | add       |
//                | transport |             | transport |
//                | to room   |             | to room   |
//                |           |             |           |
//     peer leave | remove    |  destroy    | remove    |
//    ----------> | transport | ----------> | transport |
//                | from room |             | from room |
//                |           |             |           |
//                | close     |  destroy    | close     |
//                | transport | <---------- | transport |
//                +-----------+             +-----------+
//
type controlEventType int

const (
	// controlEventTypeUnknown is the unitialized value.
	controlEventTypeUnknown controlEventType = iota
	// controlEventTypeInit is sent after the first transport is created.
	controlEventTypeInit
	// controlEventTypeDestroy is sent after the transport is about to be closed.
	// After the same event is received from the remote side, the transport is
	// actually closed.
	controlEventTypeDestroy
)

type controlEvent struct {
	Type     controlEventType `json:"type"`
	StreamID string           `json:"streamId"`
}

type controlAction int

const (
	controlActionWait = iota
	controlActionAdd
	controlActionRemove
)
