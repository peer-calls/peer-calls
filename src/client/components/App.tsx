import classnames from 'classnames'
import map from 'lodash/map'
import React from 'react'
import Peer from 'simple-peer'
import { Message } from '../actions/ChatActions'
import { dismissNotification, Notification } from '../actions/NotifyActions'
import { TextMessage } from '../actions/PeerActions'
import { AddStreamPayload, removeStream } from '../actions/StreamActions'
import * as constants from '../constants'
import Chat from './Chat'
import { Media } from './Media'
import Notifications from './Notifications'
import { Side } from './Side'
import Toolbar from './Toolbar'
import Video from './Video'

export interface AppProps {
  active: string | null
  dismissNotification: typeof dismissNotification
  init: () => void
  notifications: Record<string, Notification>
  messages: Message[]
  messagesCount: number
  peers: Record<string, Peer.Instance>
  play: () => void
  sendMessage: (message: TextMessage) => void
  streams: Record<string, AddStreamPayload>
  removeStream: typeof removeStream
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
  onHangup = () => {
    this.props.removeStream(constants.ME)
  }
  render () {
    const {
      active,
      dismissNotification,
      notifications,
      messages,
      messagesCount,
      onSendFile,
      play,
      peers,
      sendMessage,
      toggleActive,
      streams,
    } = this.props

    const { videos } = this.state

    const chatVisibleClassName = classnames({
      'chat-visible': this.state.chatVisible,
    })

    return (
      <div className="app">
        <Side align='flex-end' left zIndex={2}>
          <Toolbar
            chatVisible={this.state.chatVisible}
            messagesCount={messagesCount}
            onToggleChat={this.handleToggleChat}
            onSendFile={onSendFile}
            onHangup={this.onHangup}
            stream={streams[constants.ME]}
          />
        </Side>
        <Side className={chatVisibleClassName} top zIndex={1}>
          <Notifications
            dismiss={dismissNotification}
            notifications={notifications}
          />
          <Media />
        </Side>
        <Chat
          messages={messages}
          onClose={this.handleHideChat}
          sendMessage={sendMessage}
          visible={this.state.chatVisible}
        />
        <div className={classnames('videos', chatVisibleClassName)}>
          {streams[constants.ME] && (
            <Video
              videos={videos}
              active={active === constants.ME}
              onClick={toggleActive}
              play={play}
              stream={streams[constants.ME]}
              userId={constants.ME}
              muted
              mirrored
            />
          )}

          {
            map(peers, (_, userId) => userId)
            .filter(stream => !!stream)
            .map(userId =>
              <Video
                active={userId === active}
                key={userId}
                onClick={toggleActive}
                play={play}
                stream={streams[userId]}
                userId={userId}
                videos={videos}
              />,
            )
          }
        </div>
      </div>
    )
  }
}
