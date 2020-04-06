import classnames from 'classnames'
import React from 'react'
import screenfull from 'screenfull'
import { removeStream } from '../actions/StreamActions'
import { getDesktopStream } from '../actions/MediaActions'
import { StreamWithURL } from '../reducers/streams'
import { ME, DIAL_STATE_IN_CALL, DialState } from '../constants'

const hidden = {
  display: 'none',
}

export interface ToolbarProps {
  dialState: DialState
  messagesCount: number
  stream: StreamWithURL
  desktopStream: StreamWithURL | undefined
  onToggleChat: () => void
  onGetDesktopStream: typeof getDesktopStream
  onRemoveStream: typeof removeStream
  onSendFile: (file: File) => void
  onHangup: () => void
  chatVisible: boolean
}

export interface ToolbarState {
  readMessages: number
  camDisabled: boolean
  micMuted: boolean
  fullScreenEnabled: boolean
}

export interface ToolbarButtonProps {
  className?: string
  badge?: string | number
  blink?: boolean
  onClick: () => void
  icon: string
  offIcon?: string
  on?: boolean
  title: string
}


function ToolbarButton(props: ToolbarButtonProps) {
  const { blink, on } = props
  const icon = !on && props.offIcon ? props.offIcon : props.icon

  return (
    <a
      className={classnames('button', props.className, { blink, on })}
      onClick={props.onClick}
      href='#'
    >
      <span className={classnames('icon', icon)}>
        {!!props.badge && <span className='badge'>{props.badge}</span>}
      </span>
      <span className="tooltip">{props.title}</span>
    </a>
  )
}

export default class Toolbar
extends React.PureComponent<ToolbarProps, ToolbarState> {
  file = React.createRef<HTMLInputElement>()

  constructor(props: ToolbarProps) {
    super(props)
    this.state = {
      readMessages: props.messagesCount,
      camDisabled: false,
      micMuted: false,
      fullScreenEnabled: false,
    }
  }

  handleMicClick = () => {
    const { stream } = this.props
    stream.stream.getAudioTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    this.setState({
      ...this.state,
      micMuted: !this.state.micMuted,
    })
  }
  handleCamClick = () => {
    const { stream } = this.props
    stream.stream.getVideoTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    this.setState({
      ...this.state,
      camDisabled: !this.state.camDisabled,
    })
  }
  handleFullscreenClick = () => {
    if (screenfull.isEnabled) {
      screenfull.toggle()
      this.setState({
        ...this.state,
        fullScreenEnabled: !screenfull.isFullscreen,
      })
    }
  }
  handleHangoutClick = () => {
    window.location.href = '/'
  }
  handleSendFile = () => {
    this.file.current!.click()
  }
  handleSelectFiles = (event: React.ChangeEvent<HTMLInputElement>) => {
    Array
    .from(event.target!.files!)
    .forEach(file => this.props.onSendFile(file))
  }
  handleToggleChat = () => {
    this.setState({
      readMessages: this.props.messagesCount,
    })
    this.props.onToggleChat()
  }
  handleToggleShareDesktop = () => {
    if (this.props.desktopStream) {
      this.props.onRemoveStream(ME, this.props.desktopStream.stream)
    } else {
      this.props.onGetDesktopStream().catch(() => {})
    }
  }
  render () {
    const { messagesCount, stream } = this.props
    const unreadCount = messagesCount - this.state.readMessages
    const hasUnread = unreadCount > 0

    return (
      <div className="toolbar active">
        <input
          style={hidden}
          type="file"
          multiple
          ref={this.file}
          onChange={this.handleSelectFiles}
        />

        <ToolbarButton
          badge={unreadCount}
          className='chat'
          icon='icon-question_answer'
          blink={!this.props.chatVisible && hasUnread}
          onClick={this.handleToggleChat}
          on={this.props.chatVisible}
          title='Toggle Chat'
        />

        <ToolbarButton
          className='send-file'
          icon='icon-file-text2'
          onClick={this.handleSendFile}
          title='Send File'
        />

        <ToolbarButton
          className='stream-desktop'
          icon='icon-display'
          onClick={this.handleToggleShareDesktop}
          on={!!this.props.desktopStream}
          title='Share Desktop'
        />

        {stream && (
          <React.Fragment>
            <ToolbarButton
              onClick={this.handleMicClick}
              className='mute-audio'
              on={this.state.micMuted}
              icon='icon-mic_off'
              offIcon='icon-mic'
              title='Toggle Microphone'
            />
            <ToolbarButton
              onClick={this.handleCamClick}
              className='mute-video'
              on={this.state.camDisabled}
              icon='icon-videocam_off'
              offIcon='icon-videocam'
              title='Toggle Camera'
            />
          </React.Fragment>
        )}

        <ToolbarButton
          onClick={this.handleFullscreenClick}
          className='fullscreen'
          icon='icon-fullscreen_exit'
          offIcon='icon-fullscreen'
          on={this.state.fullScreenEnabled}
          title='Toggle Fullscreen'
        />

          {this.props.dialState === DIAL_STATE_IN_CALL && (
            <ToolbarButton
              onClick={this.props.onHangup}
              className='hangup'
              icon='icon-call_end'
              title="Hang Up"
            />
          )}

      </div>
    )
  }
}
