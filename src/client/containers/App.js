import * as CallActions from '../actions/CallActions.js'
import * as NotifyActions from '../actions/NotifyActions.js'
import * as PeerActions from '../actions/PeerActions.js'
import * as StreamActions from '../actions/StreamActions.js'
import App from '../components/App.js'
import { bindActionCreators } from 'redux'
import { connect } from 'react-redux'

function mapStateToProps (state) {
  return {
    streams: state.streams,
    peers: state.peers,
    alerts: state.alerts,
    notifications: state.notifications,
    active: state.active
  }
}

function mapDispatchToProps (dispatch) {
  return {
    toggleActive: bindActionCreators(StreamActions.toggleActive, dispatch),
    sendMessage: bindActionCreators(PeerActions.sendMessage, dispatch),
    dismissAlert: bindActionCreators(NotifyActions.dismissAlert, dispatch),
    init: bindActionCreators(CallActions.init, dispatch),
    notify: bindActionCreators(NotifyActions.info, dispatch)
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
