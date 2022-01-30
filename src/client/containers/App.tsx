import { connect } from 'react-redux'
import { hangUp, init } from '../actions/CallActions'
import { sendFile, sendText } from '../actions/ChatActions'
import { getDesktopStream, play } from '../actions/MediaActions'
import { dismissNotification } from '../actions/NotifyActions'
import { sidebarHide, sidebarShow, sidebarToggle } from '../actions/SidebarActions'
import { maximize, minimizeToggle, removeLocalStream } from '../actions/StreamActions'
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
    settings: state.settings,
    sidebarVisible: state.sidebar.visible,
    sidebarPanel: state.sidebar.panel,
  }
}

const mapDispatchToProps = {
  hangUp,
  minimizeToggle,
  maximize,
  sendText,
  dismissNotification,
  getDesktopStream,
  removeLocalStream,
  init,
  sendFile,
  play,
  sidebarToggle,
  sidebarHide,
  sidebarShow,
}

export default connect(mapStateToProps, mapDispatchToProps)(App)
