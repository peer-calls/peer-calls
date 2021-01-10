package udptransport2

import (
	"fmt"
	"testing"

	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
)

func TestControlStateTracker(t *testing.T) {
	var cst controlStateTracker

	type local struct {
		event localControlEventType
		want  remoteControlEventType
	}

	type remote struct {
		event   remoteControlEventType
		want    remoteControlEventType
		wantOK  bool
		wantErr error
	}

	type testCase struct {
		descr       string
		local       *local
		remote      *remote
		wantPending remoteControlEventType
		wantState   controlState
	}

	testCases := []testCase{
		{
			descr:       "initial state",
			wantPending: remoteControlEventTypeNone,
			wantState:   controlStateClosed,
		},
		{
			descr:     "want_create",
			local:     &local{localControlEventTypeWantCreate, remoteControlEventTypeCreate},
			wantState: controlStateCreated,
		},
		{
			descr:     "close, error",
			remote:    &remote{remoteControlEventTypeClose, remoteControlEventTypeNone, false, errUnexpectedEvent},
			wantState: controlStateCreated,
		},
		{
			descr:     "close_ack, error",
			remote:    &remote{remoteControlEventTypeCloseAck, remoteControlEventTypeNone, false, errUnexpectedEvent},
			wantState: controlStateCreated,
		},
		{
			descr:     "want_create again, no changes",
			local:     &local{localControlEventTypeWantCreate, remoteControlEventTypeNone},
			wantState: controlStateCreated,
		},
		{
			descr:     "unfamiliar local event, no changes",
			local:     &local{localControlEventType(-1), remoteControlEventTypeNone},
			wantState: controlStateCreated,
		},
		{
			descr:     "want_close, do nothing yet",
			local:     &local{localControlEventTypeWantClose, remoteControlEventTypeNone},
			wantState: controlStateCreated,
		},
		{
			descr:     "no pending event to handle yet, waiting for created_ack",
			wantState: controlStateCreated,
		},
		{
			descr:       "create_ack, but immediately handle pending want_close",
			remote:      &remote{remoteControlEventTypeCreateAck, remoteControlEventTypeNone, true, nil},
			wantState:   controlStateWriteClosed,
			wantPending: remoteControlEventTypeClose,
		},
		{
			descr:       "want_create, no remote events because waiting for close_ack",
			local:       &local{localControlEventTypeWantCreate, remoteControlEventTypeNone},
			wantState:   controlStateWriteClosed,
			wantPending: remoteControlEventTypeNone,
		},
		{
			descr:     "want_close, no remote events because already closing",
			local:     &local{localControlEventTypeWantClose, remoteControlEventTypeNone},
			wantState: controlStateWriteClosed,
		},
		{
			descr:     "invalid remote create, error",
			remote:    &remote{remoteControlEventTypeCreate, remoteControlEventTypeNone, false, errUnexpectedEvent},
			wantState: controlStateWriteClosed,
		},
		{
			descr:     "close_ack, change state to closed, no pending events",
			remote:    &remote{remoteControlEventTypeCloseAck, remoteControlEventTypeNone, true, nil},
			wantState: controlStateClosed,
		},
		{
			descr:     "want_close, nothing to do",
			local:     &local{localControlEventTypeWantClose, remoteControlEventTypeNone},
			wantState: controlStateClosed,
		},
		{
			descr:     "create_ack, error",
			remote:    &remote{remoteControlEventTypeCreateAck, remoteControlEventTypeNone, false, errUnexpectedEvent},
			wantState: controlStateClosed,
		},
		{
			descr:     "close, send close_ack",
			remote:    &remote{remoteControlEventTypeClose, remoteControlEventTypeCloseAck, false, nil},
			wantState: controlStateClosed,
		},
		{
			descr:     "create",
			remote:    &remote{remoteControlEventTypeCreate, remoteControlEventTypeCreateAck, true, nil},
			wantState: controlStateAdded,
		},
		{
			descr:     "want_create, do nothing because already added",
			local:     &local{localControlEventTypeWantCreate, remoteControlEventTypeNone},
			wantState: controlStateAdded,
		},
		{
			descr:     "unknown remote event, error",
			remote:    &remote{remoteControlEventType(-1), remoteControlEventTypeNone, false, errUnexpectedEvent},
			wantState: controlStateAdded,
		},
		{
			descr:     "close",
			remote:    &remote{remoteControlEventTypeClose, remoteControlEventTypeCloseAck, true, nil},
			wantState: controlStateClosed,
		},
	}

	for i, tc := range testCases {
		descr := fmt.Sprintf("%d. %s", i, tc.descr)

		if tc.local != nil {
			got := cst.handleLocalEvent(tc.local.event)

			assert.Equal(t, tc.local.want.String(), got.String(), "tc.local.want: %s", descr)
		}

		if tc.remote != nil {
			got, gotOK, err := cst.handleRemoteEvent(tc.remote.event)

			assert.Equal(t, tc.remote.want.String(), got.String(), "tc.remote.want: %s", descr)
			assert.Equal(t, tc.remote.wantOK, gotOK, "tc.remote.wantOK: %s", descr)
			assert.Equal(t, tc.remote.wantErr, errors.Cause(err), "tc.remote.wantErr: %s", descr)
		}

		gotPending := cst.handlePendingEvent()

		assert.Equal(t, tc.wantPending.String(), gotPending.String(), "tc.wantPending: %s", descr)

		assert.Equal(t, tc.wantState.String(), cst.state.String(), "tc.wantState: %s", descr)
	}
}

func TestLocalControlEventType_String(t *testing.T) {
	for i, tc := range []struct {
		event localControlEventType
		want  string
	}{
		{localControlEventTypeNone, "none"},
		{localControlEventTypeWantClose, "want_close"},
		{localControlEventTypeWantCreate, "want_create"},
		{localControlEventType(-1), "unknown(-1)"},
	} {
		assert.Equal(t, tc.want, tc.event.String(), "test case: %d", i)
	}
}

func TestRemoteControlEventType_String(t *testing.T) {
	for i, tc := range []struct {
		event remoteControlEventType
		want  string
	}{
		{remoteControlEventTypeNone, "none"},
		{remoteControlEventTypeCreate, "create"},
		{remoteControlEventTypeCreateAck, "create_ack"},
		{remoteControlEventTypeClose, "close"},
		{remoteControlEventTypeCloseAck, "close_ack"},
		{remoteControlEventType(-1), "unknown(-1)"},
	} {
		assert.Equal(t, tc.want, tc.event.String(), "test case: %d", i)
	}
}

func TestControlState_String(t *testing.T) {
	for i, tc := range []struct {
		event controlState
		want  string
	}{
		{controlStateClosed, "closed"},
		{controlStateCreated, "created"},
		{controlStateAdded, "added"},
		{controlStateWriteClosed, "write_closed"},
		{controlState(-1), "unknown(-1)"},
	} {
		assert.Equal(t, tc.want, tc.event.String(), "test case: %d", i)
	}
}
