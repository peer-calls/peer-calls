import classnames from 'classnames'
import map from 'lodash/map'
import React from 'react'
import Peer from 'simple-peer'
import { Message } from '../actions/ChatActions'
import { dismissNotification, Notification } from '../actions/NotifyActions'
import { Message as MessageType } from '../actions/PeerActions'
import { removeStream } from '../actions/StreamActions'
import * as constants from '../constants'
import Chat from './Chat'
import { Media } from './Media'
import Notifications from './Notifications'
import { Side } from './Side'
import Toolbar from './Toolbar'
import Video from './Video'
import { getDesktopStream } from '../actions/MediaActions'
import { StreamsState } from '../reducers/streams'
import { Nicknames } from '../reducers/nicknames'
import { getNickname } from '../nickname'

export interface AppProps {
  active: string | null
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
  toggleActive: (userId: string) => void
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
    localStreams.streams.forEach(s => {
      this.props.removeStream(constants.ME, s.stream)
    })
  }
  getLocalStreams() {
    return this.props.streams[constants.ME] || {
      userId: constants.ME,
      streams: [],
    }
  }
  render () {
    const {
      active,
      dismissNotification,
      notifications,
      nicknames,
      messages,
      messagesCount,
      onSendFile,
      play,
      peers,
      sendMessage,
      toggleActive,
      streams,
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
            stream={
              localStreams.streams
              .filter(s => s.type === constants.STREAM_TYPE_CAMERA)[0]
            }
            desktopStream={
              localStreams.streams
              .filter(s => s.type === constants.STREAM_TYPE_DESKTOP)[0]
            }
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
        <div className={classnames('videos', chatVisibleClassName)}>
          {localStreams.streams.map((s, i) => {
            const key = localStreams.userId + '_' + i
            return (
              <Video
                key={key}
                active={active === key}
                onClick={toggleActive}
                play={play}
                stream={s}
                userId={key}
                muted
                mirrored={s.type === 'camera'}
                nickname={getNickname(nicknames, localStreams.userId)}
                onChangeNickname={this.props.sendMessage}
                localUser
              />
            )
          })}
          {
            map(peers, (_, userId) => userId)
            .filter(stream => !!stream)
            .map(userId => streams[userId])
            .filter(userStreams => !!userStreams)
            .map(userStreams => {
              return userStreams.streams.map((s, i) => {
                const key = userStreams.userId + '_' + i
                return (
                  <Video
                    active={key === active}
                    key={key}
                    onClick={toggleActive}
                    play={play}
                    stream={s}
                    userId={key}
                    nickname={getNickname(nicknames, userStreams.userId)}
                    onChangeNickname={this.props.sendMessage}
                  />
                )
              })
            })
          }
        </div>
      </div>
    )
  }
}
