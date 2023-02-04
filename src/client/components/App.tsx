import classnames from 'classnames'
import forEach from 'lodash/forEach'
import React from 'react'
import Peer from 'simple-peer'
import { hangUp } from '../actions/CallActions'
import { getDesktopStream } from '../actions/MediaActions'
import { dismissNotification, Notification } from '../actions/NotifyActions'
import { Panel, sidebarPanelChat } from '../actions/SidebarActions'
import { MaximizeParams, MinimizeTogglePayload, removeLocalStream, StreamTypeDesktop } from '../actions/StreamActions'
import * as constants from '../constants'
import { Message } from '../reducers/messages'
import { Nicknames } from '../reducers/nicknames'
import { SettingsState } from '../reducers/settings'
import { StreamsState } from '../reducers/streams'
import { WindowStates } from '../reducers/windowStates'
import { Media } from './Media'
import Notifications from './Notifications'
import Sidebar from './Sidebar'
import Toolbar from './Toolbar'
import Videos from './Videos'

export interface AppProps {
  dialState: constants.DialState
  dismissNotification: typeof dismissNotification
  init: () => void
  nicknames: Nicknames
  notifications: Record<string, Notification>
  messages: Message[]
  messagesCount: number
  peers: Record<string, Peer.Instance>
  play: () => void
  sendText: (message: string) => void
  streams: StreamsState
  getDesktopStream: typeof getDesktopStream
  removeLocalStream: typeof removeLocalStream
  sendFile: (file: File) => void
  windowStates: WindowStates
  maximize: (payload: MaximizeParams) => void
  minimizeToggle: (payload: MinimizeTogglePayload) => void
  hangUp: typeof hangUp
  settings: SettingsState
  sidebarVisible: boolean
  sidebarPanel: Panel
  sidebarToggle: () => void
  sidebarHide: () => void
  sidebarShow: (panel?: Panel) => void
}

export default class App extends React.PureComponent<AppProps> {
  componentDidMount () {
    const { init } = this.props
    init()
  }
  sidebarShowChat = () => {
    this.props.sidebarShow(sidebarPanelChat)
  }
  onHangup = () => {
    const { localStreams } = this.props.streams
    forEach(localStreams, s => {
      this.props.removeLocalStream(s!.stream, s!.type)
    })
    this.props.hangUp()
  }
  render() {
    const {
      dismissNotification,
      notifications,
      nicknames,
      messages,
      messagesCount,
      minimizeToggle,
      maximize,
      sendFile,
      sendText,
      settings,
    } = this.props

    const sidebarVisibleClassName = classnames({
      'sidebar-visible': this.props.sidebarVisible,
    })

    const { localStreams } = this.props.streams

    return (
      <div className="app">
        <Toolbar
          sidebarPanel={this.props.sidebarPanel}
          sidebarVisible={this.props.sidebarVisible}
          dialState={this.props.dialState}
          messagesCount={messagesCount}
          nickname={nicknames[constants.ME]}
          onToggleSidebar={this.sidebarShowChat}
          onHangup={this.onHangup}
          desktopStream={localStreams[StreamTypeDesktop]}
          onGetDesktopStream={this.props.getDesktopStream}
          onRemoveLocalStream={this.props.removeLocalStream}
        />
        <Notifications
          className={sidebarVisibleClassName}
          dismiss={dismissNotification}
          notifications={notifications}
        />
        <Sidebar
          onHide={this.props.sidebarHide}
          onShow={this.props.sidebarShow}
          panel={this.props.sidebarPanel}
          visible={this.props.sidebarVisible}
          messages={messages}
          nicknames={nicknames}
          onMinimizeToggle={minimizeToggle}
          play={this.props.play}
          sendText={sendText}
          sendFile={sendFile}
        />
        <Media />
        {this.props.dialState !== constants.DIAL_STATE_HUNG_UP &&
          <Videos
            onMaximize={maximize}
            onMinimizeToggle={minimizeToggle}
            play={this.props.play}
            showMinimizedToolbar={settings.showMinimizedToolbar}
          />
        }
      </div>
    )
  }
}
