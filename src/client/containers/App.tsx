import { connect } from 'react-redux'
import { hangUp, init } from '../actions/CallActions'
import { sendFile, sendText } from '../actions/ChatActions'
import { getDesktopStream, play } from '../actions/MediaActions'
import { dismissNotification } from '../actions/NotifyActions'
import { minimizeToggle, removeLocalStream } from '../actions/StreamActions'
import App from '../components/App'
import { State } from '../store'

function mapStateToProps (state: State) {
  return {
    dialState: state.media.dialState,
    streams: state.streams,
    peers: state.peers,
    notifications: state.notifications,
    nicknames: state.nicknames,
    messages: state.messages.list,
    messagesCount: state.messages.count,
    windowStates: state.windowStates,
  }
}

const mapDispatchToProps = {
  hangUp,
  minimizeToggle,
  sendText,
  dismissNotification,
  getDesktopStream,
  removeLocalStream,
  init,
  sendFile,
  play,
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
