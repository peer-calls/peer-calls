import classnames from 'classnames'
import React from 'react'
import screenfull from 'screenfull'
import { getDesktopStream } from '../actions/MediaActions'
import { removeLocalStream } from '../actions/StreamActions'
import { DialState, DIAL_STATE_IN_CALL } from '../constants'
import { LocalStream } from '../reducers/streams'
import { callId } from '../window'
import { MdScreenShare, MdStopScreenShare, MdMic, MdMicOff, MdCallEnd, MdVideocam, MdVideocamOff, MdFullscreenExit, MdFullscreen, MdContentCopy, MdQuestionAnswer } from 'react-icons/md'
import { IconType } from 'react-icons'

export interface ToolbarProps {
  dialState: DialState
  nickname: string
  messagesCount: number
  cameraStream: LocalStream | undefined
  desktopStream: LocalStream | undefined
  onToggleChat: () => void
  onGetDesktopStream: typeof getDesktopStream
  onRemoveLocalStream: typeof removeLocalStream
  onHangup: () => void
  chatVisible: boolean
}

export interface ToolbarState {
  hidden: boolean
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
  icon: IconType
  offIcon?: IconType
  on?: boolean
  title: string
}

function ToolbarButton(props: ToolbarButtonProps) {
  const { blink, on } = props
  const Icon: IconType = !on && props.offIcon ? props.offIcon : props.icon

  return (
    <a
      className={classnames('button', props.className, { blink, on })}
      onClick={props.onClick}
      href='#'
    >
      <span className='icon'>
        <Icon />
        {!!props.badge && <span className='badge'>{props.badge}</span>}
      </span>
      <span className='tooltip'>{props.title}</span>
    </a>
  )
}

export default class Toolbar extends React.PureComponent<
  ToolbarProps,
  ToolbarState
> {

  constructor(props: ToolbarProps) {
    super(props)
    this.state = {
      hidden: false,
      readMessages: props.messagesCount,
      camDisabled: false,
      micMuted: false,
      fullScreenEnabled: false,
    }
  }
  componentDidMount() {
    document.body.addEventListener('click', this.toggleHidden)
    screenfull.isEnabled && screenfull.on('change', this.fullscreenChange)
  }
  componentDidWillUnmount() {
    document.body.removeEventListener('click', this.toggleHidden)
    screenfull.isEnabled && screenfull.off('change', this.fullscreenChange)
  }
  fullscreenChange = () => {
    this.setState({
      fullScreenEnabled: screenfull.isEnabled && screenfull.isFullscreen,
    })
  }
  toggleHidden = (e: MouseEvent) => {
    const t = e.target && (e.target as HTMLElement).tagName

    if (t === 'DIV' || t === 'VIDEO') {
      this.setState({ hidden: !this.state.hidden })
    }
  }
  handleMicClick = () => {
    const { cameraStream } = this.props
    if (cameraStream) {
      cameraStream.stream.getAudioTracks().forEach((track) => {
        track.enabled = !track.enabled
      })
      this.setState({
        ...this.state,
        micMuted: !this.state.micMuted,
      })
    }
  }
  handleCamClick = () => {
    const { cameraStream } = this.props
    if (cameraStream) {
      cameraStream.stream.getVideoTracks().forEach((track) => {
        track.enabled = !track.enabled
      })
      this.setState({
        ...this.state,
        camDisabled: !this.state.camDisabled,
      })
    }
  }

  handleFullscreenClick = () => {
    if (screenfull.isEnabled) {
      screenfull.toggle()
    }
  }
  handleHangoutClick = () => {
    window.location.href = '/'
  }
  copyInvitationURL = async () => {
    const { nickname } = this.props
    const link = location.href
    const text = `${nickname} has invited you for a meeting on Peer Calls. ` +
        `\nRoom: ${callId} \nLink: ${link}`
    await navigator.clipboard.writeText(text)
  }
  handleToggleChat = () => {
    this.setState({
      readMessages: this.props.messagesCount,
    })
    this.props.onToggleChat()
  }
  handleToggleShareDesktop = () => {
    if (this.props.desktopStream) {
      const { stream, type } = this.props.desktopStream
      this.props.onRemoveLocalStream(stream, type)
    } else {
      this.props.onGetDesktopStream().catch(() => {})
    }
  }
  render() {
    const { messagesCount, cameraStream } = this.props
    const unreadCount = messagesCount - this.state.readMessages
    const hasUnread = unreadCount > 0
    const isInCall = this.props.dialState === DIAL_STATE_IN_CALL

    const className = classnames('toolbar', {
      'toolbar-hidden': this.state.hidden,
    })

    return (
      <React.Fragment>
        <div className={'toolbar-other ' + className}>
          <ToolbarButton
            className='copy-url'
            key='copy-url'
            icon={MdContentCopy}
            onClick={this.copyInvitationURL}
            title='Copy Invitation URL'
          />
          {isInCall && (
            <ToolbarButton
              badge={unreadCount}
              className='chat'
              key='chat'
              icon={MdQuestionAnswer}
              blink={!this.props.chatVisible && hasUnread}
              onClick={this.handleToggleChat}
              on={this.props.chatVisible}
              title='Toggle Chat'
            />
          )}
        </div>

        <div className={'toolbar-call ' + className}>
          {isInCall && (
            <ToolbarButton
              className='stream-desktop'
              icon={MdStopScreenShare}
              offIcon={MdScreenShare}
              onClick={this.handleToggleShareDesktop}
              on={!!this.props.desktopStream}
              key='stream-desktop'
              title='Share Desktop'
            />
          )}

          {cameraStream && (
            <ToolbarButton
              onClick={this.handleMicClick}
              className='mute-audio'
              key='mute-audio'
              on={this.state.micMuted}
              icon={MdMicOff}
              offIcon={MdMic}
              title='Toggle Microphone'
            />
          )}

          {isInCall && (
            <ToolbarButton
              onClick={this.props.onHangup}
              key='hangup'
              className='hangup'
              icon={MdCallEnd}
              title='Hang Up'
            />
          )}

          {cameraStream && (
            <ToolbarButton
              onClick={this.handleCamClick}
              className='mute-video'
              key='mute-video'
              on={this.state.camDisabled}
              icon={MdVideocamOff}
              offIcon={MdVideocam}
              title='Toggle Camera'
            />
          )}

          {isInCall && (
            <ToolbarButton
              onClick={this.handleFullscreenClick}
              className='fullscreen'
              key='fullscreen'
              icon={MdFullscreenExit}
              offIcon={MdFullscreen}
              on={this.state.fullScreenEnabled}
              title='Toggle Fullscreen'
            />
          )}

        </div>
      </React.Fragment>
    )
  }
}
