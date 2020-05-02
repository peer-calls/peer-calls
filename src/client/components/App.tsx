import classnames from 'classnames'
import forEach from 'lodash/forEach'
import React from 'react'
import Peer from 'simple-peer'
import { hangUp } from '../actions/CallActions'
import { Message } from '../actions/ChatActions'
import { getDesktopStream } from '../actions/MediaActions'
import { dismissNotification, Notification } from '../actions/NotifyActions'
import { Message as MessageType } from '../actions/PeerActions'
import { MinimizeTogglePayload, removeLocalStream, StreamTypeCamera, StreamTypeDesktop } from '../actions/StreamActions'
import * as constants from '../constants'
import { Nicknames } from '../reducers/nicknames'
import { StreamsState } from '../reducers/streams'
import { WindowStates } from '../reducers/windowStates'
import Chat from './Chat'
import { Media } from './Media'
import Notifications from './Notifications'
import { Side } from './Side'
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
  sendMessage: (message: MessageType) => void
  streams: StreamsState
  getDesktopStream: typeof getDesktopStream
  removeLocalStream: typeof removeLocalStream
  onSendFile: (file: File) => void
  windowStates: WindowStates
  minimizeToggle: (payload: MinimizeTogglePayload) => void
  hangUp: typeof hangUp
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
    const { localStreams } = this.props.streams
    forEach(localStreams, s => {
      this.props.removeLocalStream(s!.stream, s!.type)
    })
    this.props.hangUp()
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

    const { localStreams } = this.props.streams

    return (
      <div className="app">
        <Side align='flex-end' left zIndex={2}>
          <Toolbar
            chatVisible={this.state.chatVisible}
            dialState={this.props.dialState}
            messagesCount={messagesCount}
            nickname={nicknames[constants.ME]}
            onToggleChat={this.handleToggleChat}
            onSendFile={onSendFile}
            onHangup={this.onHangup}
            cameraStream={localStreams[StreamTypeCamera]}
            desktopStream={localStreams[StreamTypeDesktop]}
            onGetDesktopStream={this.props.getDesktopStream}
            onRemoveLocalStream={this.props.removeLocalStream}
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
          onMinimizeToggle={this.props.minimizeToggle}
          streams={this.props.streams}
          play={this.props.play}
          nicknames={this.props.nicknames}
          windowStates={this.props.windowStates}
        />
      </div>
    )
  }
}
