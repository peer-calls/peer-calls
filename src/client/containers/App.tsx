import { connect } from 'react-redux'
import { init } from '../actions/CallActions'
import { getDesktopStream, play } from '../actions/MediaActions'
import { dismissNotification } from '../actions/NotifyActions'
import { sendFile, sendMessage } from '../actions/PeerActions'
import { toggleActive, removeStream } from '../actions/StreamActions'
import App from '../components/App'
import { State } from '../store'

function mapStateToProps (state: State) {
  return {
    streams: state.streams,
    peers: state.peers,
    notifications: state.notifications,
    nicknames: state.nicknames,
    messages: state.messages.list,
    messagesCount: state.messages.count,
    active: state.active,
  }
}

const mapDispatchToProps = {
  toggleActive,
  sendMessage,
  dismissNotification,
  getDesktopStream,
  removeStream,
  init,
  onSendFile: sendFile,
  play,
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
