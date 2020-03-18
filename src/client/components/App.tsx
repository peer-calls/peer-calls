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
import { StreamsState, StreamWithURL } from '../reducers/streams'
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
  private getLocalStreams() {
    return this.props.streams[constants.ME] || {
      userId: constants.ME,
      streams: [],
    }
  }
  private getVideoStreams() {
    const {active, peers, streams} = this.props
    const localStreams = this.getLocalStreams()

    type s = {
      key: string
      stream: StreamWithURL
      userId: string
      muted?: boolean
      localUser?: boolean
      mirrored?: boolean
    }
    let activeStream: s | undefined
    const otherStreams: Array<s> = []

    function addStream(s: s) {
      if (active === s.key) {
        activeStream = s
      } else {
        otherStreams.push(s)
      }
    }

    localStreams.streams.map((stream, i) => {
      const key = localStreams.userId + '_' + i
      addStream({
        key,
        stream,
        userId: localStreams.userId,
        mirrored: stream.type === 'camera',
        muted: true,
        localUser: true,
      })
    })

    map(peers, (_, userId) => userId)
    .filter(stream => !!stream)
    .map(userId => streams[userId])
    .filter(userStreams => !!userStreams)
    .map(userStreams => {
      return userStreams.streams.map((stream, i) => {
        const key = userStreams.userId + '_' + i
        addStream({
          key,
          stream,
          userId: userStreams.userId,
        })
      })
    })

    return { activeStream, localStreams, otherStreams }
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
      sendMessage,
      toggleActive,
    } = this.props

    const chatVisibleClassName = classnames({
      'chat-visible': this.state.chatVisible,
    })

    const { activeStream, localStreams, otherStreams } = this.getVideoStreams()

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

        {activeStream && (
          <Video
            key={activeStream.key}
            active={active === activeStream.key}
            onClick={toggleActive}
            play={play}
            stream={activeStream.stream}
            userId={activeStream.key}
            muted={activeStream.muted}
            mirrored={activeStream.mirrored}
            nickname={getNickname(nicknames, activeStream.userId)}
            onChangeNickname={this.props.sendMessage}
            localUser={activeStream.localUser}
          />
        )}
        <div className={classnames('videos', chatVisibleClassName)}>
          {otherStreams.map(stream => {
            return (
              <Video
                key={stream.key}
                active={active === stream.key}
                onClick={toggleActive}
                play={play}
                stream={stream.stream}
                userId={stream.key}
                muted={stream.muted}
                mirrored={stream.mirrored}
                nickname={getNickname(nicknames, stream.userId)}
                onChangeNickname={this.props.sendMessage}
                localUser={stream.localUser}
              />
            )
          })}
        </div>
      </div>
    )
  }
}
