package udptransport2

import (
	"fmt"

	"github.com/juju/errors"
)

// transportTracker ensures that local and remote transports are both created
// before adding them to the rooms. It also ensures that the remote one is
// removed before closing the transport.
type transportTracker struct {
	transport *Transport

	state controlState

	pendingLocalEvent controlEventType
}

func (t *transportTracker) setTransport(transport *Transport) {
	t.transport = transport
}

func (t *transportTracker) wantCreate() controlEventType {
	switch t.state {
	case controlStateClosed:
		t.state = controlStateCreated

		return controlEventTypeCreate
	case controlStateCreated:
		// Already in the process of creating, nothing else to do.
		t.pendingLocalEvent = controlEventTypeNone
	case controlStateAdded:
		// Already created, nothing else to do.
		t.pendingLocalEvent = controlEventTypeNone
	case controlStateWriteClosed:
		// Wait for close_ack event first.
		t.pendingLocalEvent = controlEventTypeCreate
	}

	return controlEventTypeNone
}

func (t *transportTracker) wantClose() controlEventType {
	switch t.state {
	case controlStateClosed:
		// Already closed, nothing else to do.
		t.pendingLocalEvent = controlEventTypeNone
	case controlStateCreated:
		// Wait for create_ack event first.
		t.pendingLocalEvent = controlEventTypeClose
	case controlStateAdded:
		t.pendingLocalEvent = controlEventTypeNone
		t.state = controlStateWriteClosed

		return controlEventTypeClose
	case controlStateWriteClosed:
		// Already in the process of destroying, nothing else to do.
		t.pendingLocalEvent = controlEventTypeNone
	}

	return controlEventTypeNone
}

// handlePendingEvent must be called after each call to handleRemoteEvent to
// see if there are any pending actions to perform.
func (t *transportTracker) handlePendingEvent() controlEventType {
	// nolint:exhaustive
	switch t.pendingLocalEvent {
	case controlEventTypeClose:
		return t.wantClose()
	case controlEventTypeCreate:
		return t.wantCreate()
	}

	return controlEventTypeNone
}

// handleRemoteEvent updates the internal state based on the event received.
// The first return value indicates what event to send to the remote side.
// When the state was changed, the second return value will be true. It will
// be false when the state change was unchanged.
//
// When the state was changed and the event processed, the user must call the
// handlePendingEvents to see if there are any localc events that were waiting
// to be handled.
func (t *transportTracker) handleRemoteEvent(event controlEventType) (controlEventType, bool, error) {
	oldState := t.state

	switch event {
	case controlEventTypeCreate:
		if t.state != controlStateClosed && t.state != controlStateAdded {
			// Unexpeted event.
			break
		}

		t.state = controlStateAdded

		return controlEventTypeCreateAck, oldState != t.state, nil

	case controlEventTypeCreateAck:
		if t.state != controlStateAdded {
			// Unexpected event.
			break
		}

		t.state = controlStateAdded

		return controlEventTypeNone, oldState != t.state, nil
	case controlEventTypeClose:
		if t.state != controlStateAdded && t.state != controlStateClosed {
			// Unexpected event.
			break
		}

		t.state = controlStateClosed

		return controlEventTypeCloseAck, oldState != t.state, nil
	case controlEventTypeCloseAck:
		if t.state != controlStateWriteClosed {
			// Unexpected event.
			break
		}

		t.state = controlStateClosed

		return controlEventTypeNone, oldState != t.state, nil
	case controlEventTypeNone:
	}

	return controlEventTypeNone, false, errors.Annotatef(errUnexpectedEvent, "state: %s, event: %s", t.state, event)
}

// controlEventType are the possible types of event sent between remote Peer
// Calls nodes to initiate transport.
//
//                   node 1                  node 2
//                +---------------------+              +------------------+
//                | (0)                 |              | (0)              |
//                | state: closed       |              | state: closed    |
//     peer join  |                     |              |                  |
//    ----------> | (1)                 |              |                  |
//                | create transport    |    create    |                  |
//                | state: created      | -----------> | (2)              |
//                |                     |              | create transport |
//                |                     |  create_ack  | add to room      |
//                | (3)                 | <----------- | state: added     |
//                | add to room         |              |                  |
//                | state: added        |              |                  |
//                |                     |              |                  |
//     peer leave |                     |              |                  |
//     ---------> | (4)                 |              |                  |
//                | close write         |    close     |                  |
//                | state: write_closed | -----------> | (5)              |
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
type controlEventType int

const (
	// controlEventTypeNone is the unitialized value.
	controlEventTypeNone controlEventType = iota
	// controlEventTypeCreate is sent after the first transport is created.
	controlEventTypeCreate
	// controlEventTypeCreateAck is sent to acknowledge the create event was handled.
	controlEventTypeCreateAck
	// controlEventTypeClose is sent after the transport is about to be closed.
	// After the same event is received from the remote side, the transport is
	// actually closed.
	controlEventTypeClose
	// controlEventTypeCloseAck is sent to acknowledge the destroy event was
	// handled.
	controlEventTypeCloseAck
)

func (c controlEventType) String() string {
	switch c {
	case controlEventTypeNone:
		return "none"
	case controlEventTypeCreate:
		return "create"
	case controlEventTypeCreateAck:
		return "create_ack"
	case controlEventTypeClose:
		return "close"
	case controlEventTypeCloseAck:
		return "close_ack"
	default:
		return fmt.Sprintf("unknown(%d)", c)
	}
}

type controlEvent struct {
	Type     controlEventType `json:"type"`
	StreamID string           `json:"streamId"`
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
