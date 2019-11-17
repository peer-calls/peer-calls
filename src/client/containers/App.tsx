import { connect } from 'react-redux'
import { init } from '../actions/CallActions'
import { play } from '../actions/MediaActions'
import { dismissAlert, dismissNotification } from '../actions/NotifyActions'
import { sendFile, sendMessage } from '../actions/PeerActions'
import { toggleActive } from '../actions/StreamActions'
import App from '../components/App'
import { State } from '../store'

function mapStateToProps (state: State) {
  return {
    streams: state.streams,
    peers: state.peers,
    alerts: state.alerts,
    notifications: state.notifications,
    messages: state.messages.list,
    messagesCount: state.messages.count,
    active: state.active,
  }
}

const mapDispatchToProps = {
  toggleActive,
  sendMessage,
  dismissAlert: dismissAlert,
  dismissNotification,
  init,
  onSendFile: sendFile,
  play,
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
