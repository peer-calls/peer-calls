import * as CallActions from '../actions/CallActions'
import * as NotifyActions from '../actions/NotifyActions'
import * as PeerActions from '../actions/PeerActions'
import * as StreamActions from '../actions/StreamActions'
import App from '../components/App'
import { bindActionCreators, Dispatch } from 'redux'
import { connect } from 'react-redux'
import { State } from '../store'

function mapStateToProps (state: State) {
  return {
    streams: state.streams,
    peers: state.peers,
    alerts: state.alerts,
    notifications: state.notifications,
    messages: state.messages,
    active: state.active,
  }
}

function mapDispatchToProps (dispatch: Dispatch) {
  return {
    toggleActive: bindActionCreators(StreamActions.toggleActive, dispatch),
    sendMessage: bindActionCreators(PeerActions.sendMessage, dispatch),
    dismissAlert: bindActionCreators(NotifyActions.dismissAlert, dispatch),
    init: bindActionCreators(CallActions.init, dispatch),
    onSendFile: bindActionCreators(PeerActions.sendFile, dispatch),
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
