import map from 'lodash/map'
import React from 'react'
import Peer from 'simple-peer'
import { Message } from '../actions/ChatActions'
import { Alert, Notification } from '../actions/NotifyActions'
import { TextMessage } from '../actions/PeerActions'
import { AddStreamPayload } from '../actions/StreamActions'
import * as constants from '../constants'
import Alerts from './Alerts'
import Chat from './Chat'
import Notifications from './Notifications'
import Toolbar from './Toolbar'
import Video from './Video'
import { Media } from './Media'

export interface AppProps {
  active: string | null
  alerts: Alert[]
  dismissAlert: (alert: Alert) => void
  init: () => void
  notifications: Record<string, Notification>
  messages: Message[]
  messagesCount: number
  peers: Record<string, Peer.Instance>
  sendMessage: (message: TextMessage) => void
  streams: Record<string, AddStreamPayload>
  onSendFile: (file: File) => void
  toggleActive: (userId: string) => void
}

export interface AppState {
  videos: Record<string, unknown>
  chatVisible: boolean
}

export default class App extends React.PureComponent<AppProps, AppState> {
  state: AppState = {
    videos: {},
    chatVisible: false,
  }
  handleShowChat = () => {
    this.setState({
      chatVisible: true,
    })
  }
  handleHideChat = () => {
    this.setState({
      chatVisible: false,
    })
  }
  handleToggleChat = () => {
    return this.state.chatVisible
      ? this.handleHideChat()
      : this.handleShowChat()
  }
  componentDidMount () {
    const { init } = this.props
    init()
  }
  render () {
    const {
      active,
      alerts,
      dismissAlert,
      notifications,
      messages,
      messagesCount,
      onSendFile,
      peers,
      sendMessage,
      toggleActive,
      streams,
    } = this.props

    const { videos } = this.state

    return (
      <div className="app">
        <Toolbar
          chatVisible={this.state.chatVisible}
          messagesCount={messagesCount}
          onToggleChat={this.handleToggleChat}
          onSendFile={onSendFile}
          stream={streams[constants.ME]}
        />
        <Alerts alerts={alerts} dismiss={dismissAlert} />
        <Notifications notifications={notifications} />
        <Media />
        <Chat
          messages={messages}
          onClose={this.handleHideChat}
          sendMessage={sendMessage}
          visible={this.state.chatVisible}
        />
        <div className="videos">
          <Video
            videos={videos}
            active={active === constants.ME}
            onClick={toggleActive}
            stream={streams[constants.ME]}
            userId={constants.ME}
            muted
            mirrored
          />

          {map(peers, (_, userId) => (
            <Video
              active={userId === active}
              key={userId}
              onClick={toggleActive}
              stream={streams[userId]}
              userId={userId}
              videos={videos}
            />
          ))}
        </div>
      </div>
    )
  }
}
