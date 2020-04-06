import classnames from 'classnames'
import keyBy from 'lodash/keyBy'
import forEach from 'lodash/forEach'
import React from 'react'
import Peer from 'simple-peer'
import { Message } from '../actions/ChatActions'
import { dismissNotification, Notification } from '../actions/NotifyActions'
import { Message as MessageType, destroyPeers } from '../actions/PeerActions'
import { removeStream, MinimizeTogglePayload } from '../actions/StreamActions'
import * as constants from '../constants'
import Chat from './Chat'
import { Media } from './Media'
import Notifications from './Notifications'
import { Side } from './Side'
import Toolbar from './Toolbar'
import Videos from './Videos'
import { getDesktopStream } from '../actions/MediaActions'
import { StreamsState } from '../reducers/streams'
import { WindowStates } from '../reducers/windowStates'
import { Nicknames } from '../reducers/nicknames'

export interface AppProps {
  dismissNotification: typeof dismissNotification
  init: () => void
  nicknames: Nicknames
  notifications: Record<string, Notification>
  messages: Message[]
  messagesCount: number
  peers: Record<string, Peer.Instance>
  play: () => void
  sendMessage: (message: MessageType) => void
  streams: StreamsState
  getDesktopStream: typeof getDesktopStream
  removeStream: typeof removeStream
  onSendFile: (file: File) => void
  windowStates: WindowStates
  minimizeToggle: (payload: MinimizeTogglePayload) => void
  destroyPeers: typeof destroyPeers
}

export interface AppState {
  chatVisible: boolean
}

export default class App extends React.PureComponent<AppProps, AppState> {
  state: AppState = {
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
    const localStreams = this.getLocalStreams()
    forEach(localStreams, s => {
      this.props.removeStream(constants.ME, s.stream)
    })
    this.props.destroyPeers()
  }
  getLocalStreams() {
    const ls = this.props.streams[constants.ME]
    return ls ? keyBy(ls.streams, 'type') : {}
  }
  render () {
    const {
      dismissNotification,
      notifications,
      nicknames,
      messages,
      messagesCount,
      onSendFile,
      sendMessage,
    } = this.props

    const chatVisibleClassName = classnames({
      'chat-visible': this.state.chatVisible,
    })

    const localStreams = this.getLocalStreams()

    return (
      <div className="app">
        <Side align='flex-end' left zIndex={2}>
          <Toolbar
            chatVisible={this.state.chatVisible}
            messagesCount={messagesCount}
            onToggleChat={this.handleToggleChat}
            onSendFile={onSendFile}
            onHangup={this.onHangup}
            stream={localStreams[constants.STREAM_TYPE_CAMERA]}
            desktopStream={localStreams[constants.STREAM_TYPE_DESKTOP]}
            onGetDesktopStream={this.props.getDesktopStream}
            onRemoveStream={this.props.removeStream}
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
          nicknames={nicknames}
          onClose={this.handleHideChat}
          sendMessage={sendMessage}
          visible={this.state.chatVisible}
        />

        <Videos
          onChangeNickname={sendMessage}
          onMinimizeToggle={this.props.minimizeToggle}
          streams={this.props.streams}
          play={this.props.play}
          nicknames={this.props.nicknames}
          peers={this.props.peers}
          windowStates={this.props.windowStates}
        />
      </div>
    )
  }
}
