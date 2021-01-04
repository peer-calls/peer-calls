package udptransport2

// import (
// 	"fmt"
// 	"testing"

// 	"github.com/juju/errors"
// 	"github.com/stretchr/testify/assert"
// )

// func TestTransportStateTracker(t *testing.T) {
// 	var tracker transportStateTracker

// 	type testCase struct {
// 		descr     string
// 		action    func() error
// 		wantState transportState
// 		wantErr   error
// 	}

// 	testCases := []testCase{
// 		{"initial state", nil, transportStateDisconnected, nil},

// 		{"local abort", tracker.localAbort, transportStateDisconnected, errInvalidStateTransition},
// 		{"remote abort", tracker.remoteAbort, transportStateDisconnected, errInvalidStateTransition},

// 		{"local request", tracker.localRequest, transportStateLocalRequested, nil},
// 		{"local request", tracker.localRequest, transportStateLocalRequested, errInvalidStateTransition},
// 		{"local abort", tracker.localAbort, transportStateDisconnected, nil},

// 		{"remote request", tracker.remoteRequest, transportStateRemoteRequested, nil},
// 		{"remote abort", tracker.remoteAbort, transportStateDisconnected, nil},

// 		{"local request", tracker.localRequest, transportStateLocalRequested, nil},
// 		{"remote request", tracker.remoteRequest, transportStateConnecting, nil},
// 		{"remote abort", tracker.remoteAbort, transportStateDisconnecting, nil},
// 		{"disconnect", tracker.disconnect, transportStateDisconnected, nil},

// 		{"disconnect", tracker.disconnect, transportStateDisconnected, errInvalidStateTransition},
// 		{"connect success", tracker.connectSuccess, transportStateDisconnected, errInvalidStateTransition},
// 		{"connect failed", tracker.connectFailed, transportStateDisconnected, errInvalidStateTransition},
// 		{"remote request", tracker.remoteRequest, transportStateRemoteRequested, nil},
// 		{"local request", tracker.localRequest, transportStateConnecting, nil},
// 		{"local abort", tracker.localAbort, transportStateDisconnecting, nil},
// 		{"disconnect", tracker.disconnect, transportStateDisconnected, nil},

// 		{"remote request", tracker.remoteRequest, transportStateRemoteRequested, nil},
// 		{"local request", tracker.localRequest, transportStateConnecting, nil},
// 		{"connect failed", tracker.connectFailed, transportStateDisconnecting, nil},
// 		{"disconnect", tracker.disconnect, transportStateDisconnected, nil},

// 		{"remote request", tracker.remoteRequest, transportStateRemoteRequested, nil},
// 		{"local request", tracker.localRequest, transportStateConnecting, nil},
// 		{"connect success", tracker.connectSuccess, transportStateConnected, nil},
// 		{"connect failed", tracker.connectFailed, transportStateDisconnecting, nil},
// 		{"disconnect", tracker.disconnect, transportStateDisconnected, nil},

// 		{"remote request", tracker.remoteRequest, transportStateRemoteRequested, nil},
// 		{"local request", tracker.localRequest, transportStateConnecting, nil},
// 		{"connect success", tracker.connectSuccess, transportStateConnected, nil},
// 		{"local abort", tracker.localAbort, transportStateDisconnecting, nil},
// 		{"disconnect", tracker.disconnect, transportStateDisconnected, nil},

// 		{"remote request", tracker.remoteRequest, transportStateRemoteRequested, nil},
// 		{"local request", tracker.localRequest, transportStateConnecting, nil},
// 		{"connect success", tracker.connectSuccess, transportStateConnected, nil},
// 		{"remote abort", tracker.remoteAbort, transportStateDisconnecting, nil},
// 		{"disconnect", tracker.disconnect, transportStateDisconnected, nil},
// 	}

// 	for i, tc := range testCases {
// 		descr := fmt.Sprintf("%d. %s", i, tc.descr)

// 		var err error

// 		if tc.action != nil {
// 			err = tc.action()
// 		}

// 		assert.Equal(t, tc.wantErr, errors.Cause(err), "want err: %s", descr)
// 		assert.Equal(t, tc.wantState, tracker.state, "want state: %s", descr)
