package udptransport2

//// transportState describes the transport state. Below is the state diagram.
////
////        /------------------------------------------------------------\
////        |                             disconnect                     |
////        |                                                           +---------------+
////        |   /-------------------\                                   | disconnecting |
////        |   |   local aborts    |                                   +---------------+
////        |   |                   |                                   ^   local abort ^
////        |   |   +  local    +------------------+  remote            |  remote abort |
////        v   v /------------>| local requested  |-----------v        |               |
//// +--------------+ request   +------------------+ request  +------------+         +-----------+
//// | disconnected |                                         | connecting |-------->| connected |
//// +--------------+ remote    +------------------+  local   +------------+ connect +-----------+
////            ^ \------------>| remote requested |-----------^
////            |    request    +------------------+ request
////            |                   |
////            |  remote abort     |
////            \-------------------/
//type transportState int

//const (
//	transportStateDisconnected transportState = iota
//	transportStateRemoteRequested
//	transportStateLocalRequested
//	transportStateConnecting
//	transportStateConnected
//	transportStateDisconnecting
//)

//type transportStateTracker struct {
//	state transportState
//}

//var errInvalidStateTransition = errors.New("invalid state transition")

//func (t *transportStateTracker) localRequest() error {
//	switch t.state {
//	case transportStateDisconnected:
//		t.state = transportStateLocalRequested
//	case transportStateRemoteRequested:
//		t.state = transportStateConnecting
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "local request: state: %d", t.state)
//	}

//	return nil
//}

//func (t *transportStateTracker) localAbort() error {
//	switch t.state {
//	case transportStateLocalRequested:
//		t.state = transportStateDisconnected
//	case transportStateConnecting, transportStateConnected:
//		t.state = transportStateDisconnecting
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "local abort: state: %d", t.state)
//	}

//	return nil
//}

//func (t *transportStateTracker) remoteRequest() error {
//	switch t.state {
//	case transportStateDisconnected:
//		t.state = transportStateRemoteRequested
//	case transportStateLocalRequested:
//		t.state = transportStateConnecting
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "remote request: state: %d", t.state)
//	}

//	return nil
//}

//func (t *transportStateTracker) remoteAbort() error {
//	switch t.state {
//	case transportStateRemoteRequested:
//		t.state = transportStateDisconnected
//	case transportStateConnecting, transportStateConnected:
//		t.state = transportStateDisconnecting
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "local abort: state: %d", t.state)
//	}

//	return nil
//}

//func (t *transportStateTracker) connectSuccess() error {
//	switch t.state {
//	case transportStateConnecting:
//		t.state = transportStateConnected
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "connect success: state: %d", t.state)
//	}

//	return nil
//}

//func (t *transportStateTracker) connectFailed() error {
//	switch t.state {
//	case transportStateConnecting, transportStateConnected:
//		t.state = transportStateDisconnecting
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "connect failed: state: %d", t.state)
//	}

//	return nil
//}

//func (t *transportStateTracker) disconnect() error {
//	switch t.state {
//	case transportStateDisconnecting:
//		t.state = transportStateDisconnected
//	default:
//		return errors.Annotatef(errInvalidStateTransition, "disconnect: state: %d", t.state)
//	}

//	return nil
//}
