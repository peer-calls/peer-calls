package udptransport2

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/identifiers"
)

// controlStateTracker ensures that local and remote transports are both
// created before adding them to the rooms. It also ensures that the remote one
// is removed before closing the transport.
//
// The initial state is closed.
//
//     +--------+                                close_ack
//     | closed |<------------------------\<--------------------\
//     +--------+                          |                    |
//       |    |  want_create  +---------+  |             +--------------+
//       |    \-------------->| created |  | close_ack   | write_closed |
//       |                    +---------+  |             +--------------+
//       |                             |   |                    |
//       |                  create_ack |   |                    |
//       |                             v   |                    |
//       |           create           +---------+   want_close  |
//       \--------------------------->|  added  |---------------/
//                                    +---------+
//
// Local events (want_create and want_close) received during the waiting for
// acknowledgement (states created and write_closed) are stored and should be
// retried afterwards by calling handleLocalEvent. Calling wantClose quickly
// after wantCreate will replace the pending state.
//
// See documentation of remoteControlEventType for more details.
//
// Note: there could potentially be a small timing issue after the remote
// create event is received and before the create_ack event is sent back to
// the remote node: the transport will be added to the room before the remote
// node adds the transport so it might start sending data to the remote node
// before the remote node is ready to receive the data. But if the create_ack
// is sent first and the transport immediately added to the room, this should
// be a very short period. It might be better to test this first and then
// optimize if necessary.
type controlStateTracker struct {
	state controlState

	pendingLocalEvent remoteControlEventType
}

func (t *controlStateTracker) wantCreate() remoteControlEventType {
	switch t.state {
	case controlStateClosed:
		t.state = controlStateCreated

		return remoteControlEventTypeCreate
	case controlStateCreated:
		// Already in the process of creating, nothing else to do.
		t.pendingLocalEvent = remoteControlEventTypeNone
	case controlStateAdded:
		// Already created, nothing else to do.
		t.pendingLocalEvent = remoteControlEventTypeNone
	case controlStateWriteClosed:
		// Wait for close_ack event first.
		t.pendingLocalEvent = remoteControlEventTypeCreate
	}

	return remoteControlEventTypeNone
}

func (t *controlStateTracker) wantClose() remoteControlEventType {
	switch t.state {
	case controlStateClosed:
		// Already closed, nothing else to do.
		t.pendingLocalEvent = remoteControlEventTypeNone
	case controlStateCreated:
		// Wait for create_ack event first.
		t.pendingLocalEvent = remoteControlEventTypeClose
	case controlStateAdded:
		t.pendingLocalEvent = remoteControlEventTypeNone
		t.state = controlStateWriteClosed

		return remoteControlEventTypeClose
	case controlStateWriteClosed:
		// Already in the process of destroying, nothing else to do.
		t.pendingLocalEvent = remoteControlEventTypeNone
	}

	return remoteControlEventTypeNone
}

func (t *controlStateTracker) handleLocalEvent(event localControlEventType) remoteControlEventType {
	// nolint:exhaustive
	switch event {
	case localControlEventTypeWantClose:
		return t.wantClose()
	case localControlEventTypeWantCreate:
		return t.wantCreate()
	}

	return remoteControlEventTypeNone
}

// handlePendingEvent must be called after each call to handleRemoteEvent to
// see if there are any pending actions to perform.
func (t *controlStateTracker) handlePendingEvent() remoteControlEventType {
	// nolint:exhaustive
	switch t.pendingLocalEvent {
	case remoteControlEventTypeClose:
		return t.wantClose()
	case remoteControlEventTypeCreate:
		return t.wantCreate()
	}

	return remoteControlEventTypeNone
}

// handleRemoteEvent updates the internal state based on the event received.
// The first return value indicates what event to send to the remote side.
// When the state was changed, the second return value will be true. It will
// be false when the state change was unchanged.
//
// When the state was changed and the event processed, the user must call the
// handlePendingEvents to see if there are any localc events that were waiting
// to be handled.
func (t *controlStateTracker) handleRemoteEvent(event remoteControlEventType) (remoteControlEventType, bool, error) {
	prevState := t.state

	switch event {
	case remoteControlEventTypeCreate:
		if t.state != controlStateClosed && t.state != controlStateAdded {
			// Unexpeted event.
			break
		}

		t.state = controlStateAdded

		return remoteControlEventTypeCreateAck, t.state != prevState, nil

	case remoteControlEventTypeCreateAck:
		if t.state != controlStateCreated {
			// Unexpected event.
			break
		}

		t.state = controlStateAdded

		return remoteControlEventTypeNone, t.state != prevState, nil
	case remoteControlEventTypeClose:
		if t.state != controlStateAdded && t.state != controlStateClosed {
			// Unexpected event.
			break
		}

		t.state = controlStateClosed

		return remoteControlEventTypeCloseAck, t.state != prevState, nil
	case remoteControlEventTypeCloseAck:
		if t.state != controlStateWriteClosed {
			// Unexpected event.
			break
		}

		t.state = controlStateClosed

		return remoteControlEventTypeNone, t.state != prevState, nil
	case remoteControlEventTypeNone:
	}

	return remoteControlEventTypeNone, false, errors.Annotatef(errUnexpectedEvent, "state: %s, event: %s", t.state, event)
}

type localControlEventType int

const (
	localControlEventTypeNone localControlEventType = iota
	localControlEventTypeWantCreate
	localControlEventTypeWantClose
)

func (c localControlEventType) String() string {
	switch c {
	case localControlEventTypeNone:
		return "none"
	case localControlEventTypeWantCreate:
		return "want_create"
	case localControlEventTypeWantClose:
		return "want_close"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

// remoteControlEventType are the possible types of event sent between remote Peer
// Calls nodes to initiate transport.
//
//                        node 1                              node 2
//                +---------------------+              +------------------+
//                | (0)                 |              | (0)              |
//                | state: closed       |              | state: closed    |
//    want_create |                     |              |                  |
//    ----------> | (1)                 |              |                  |
//    e.g.        | create transport    |    create    |                  |
//    peer join   | state: created      | -----------> | (2)              |
//                |                     |              | create transport |
//                |                     |  create_ack  | add to room      |
//                | (3)                 | <----------- | state: added     |
//                | add to room         |              |                  |
//                | state: added        |              |                  |
//                |                     |              |                  |
//     want_close |                     |              |                  |
//     ---------> | (4)                 |              |                  |
//     e.g.       | close write         |    close     |                  |
//     peer leave | state: write_closed | -----------> | (5)              |
//                |                     |              | close transport  |
//                |                     |   close_ack  | remove transport |
//                | (6)                 | <----------- | state: closed    |
//                | close transport     |              |                  |
//                | remove transport    |              |                  |
//                | state: closed       |              |                  |
//                +---------------------+              +------------------+
//
// Once a remote control event is received, it is only applied when:
//
//   1. The latest state transition has been acked, and
//   2. Both remote and local state are the same.
//
// Transitions received before a control event is acked:
//
//   1. Receiving init event after init event was sent, but before init_ack was
//      received. Results with init_ack sent, but still wait init_ack to be
//      received from the remote end.
//   2. Receiving destroy event after destroy was sent, but before destroy_ack
//      was received. Results with destroy_ack sent, but sitll wait for
//      destroy_ack to be received.
//   3. Receiving destroy after init, but before init was acknowledged is
//      illegal and signals a bug in the code. After a first event is sent, the
//      local side must not send any events until that event is acknowledged.
//   4. Receiving init after destroy, but before destroy was acknowledged is
//      illegal and signals a bug in the code, see (3).
type remoteControlEventType int

const (
	// remoteControlEventTypeNone is the unitialized value.
	remoteControlEventTypeNone remoteControlEventType = iota
	// remoteControlEventTypeCreate is sent after the first transport is created.
	remoteControlEventTypeCreate
	// remoteControlEventTypeCreateAck is sent to acknowledge the create event was handled.
	remoteControlEventTypeCreateAck
	// remoteControlEventTypeClose is sent after the transport is about to be closed.
	// After the same event is received from the remote side, the transport is
	// actually closed.
	remoteControlEventTypeClose
	// remoteControlEventTypeCloseAck is sent to acknowledge the destroy event was
	// handled.
	remoteControlEventTypeCloseAck
)

func (c remoteControlEventType) String() string {
	switch c {
	case remoteControlEventTypeNone:
		return "none"
	case remoteControlEventTypeCreate:
		return "create"
	case remoteControlEventTypeCreateAck:
		return "create_ack"
	case remoteControlEventTypeClose:
		return "close"
	case remoteControlEventTypeCloseAck:
		return "close_ack"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

type remoteControlEvent struct {
	Type     remoteControlEventType `json:"type"`
	StreamID identifiers.RoomID     `json:"streamId"`
}

type localControlEvent struct {
	typ      localControlEventType
	streamID identifiers.RoomID
}

type controlState int

const (
	controlStateClosed controlState = iota
	controlStateCreated
	controlStateAdded
	controlStateWriteClosed
)

func (c controlState) String() string {
	switch c {
	case controlStateClosed:
		return "closed"
	case controlStateCreated:
		return "created"
	case controlStateAdded:
		return "added"
	case controlStateWriteClosed:
		return "write_closed"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

var errUnexpectedEvent = errors.New("unexpected event")
