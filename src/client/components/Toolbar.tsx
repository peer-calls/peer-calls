import React, { ReactEventHandler, ChangeEvent } from 'react'
import classnames from 'classnames'
import screenfull from 'screenfull'
import { Message } from '../actions/ChatActions'
import { AddStreamPayload } from '../actions/StreamActions'

const hidden = {
  display: 'none',
}

export interface ToolbarProps {
  messages: Message[]
  stream: AddStreamPayload
  onToggleChat: () => void
  onSendFile: (file: File) => void
  chatVisible: boolean
}

export interface ToolbarState {
  readMessages: number
  camDisabled: boolean
  micMuted: boolean
  fullScreenEnabled: boolean
}

export default class Toolbar
extends React.PureComponent<ToolbarProps, ToolbarState> {
  file = React.createRef<HTMLInputElement>()

  constructor(props: ToolbarProps) {
    super(props)
    this.state = {
      readMessages: props.messages.length,
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
        fullScreenEnabled: !this.state.fullScreenEnabled,
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
      readMessages: this.props.messages.length,
    })
    this.props.onToggleChat()
  }
  render () {
    const { messages, stream } = this.props

    return (
      <div className="toolbar active">
        <div onClick={this.handleToggleChat}
          className={classnames('button chat', {
            on: this.props.chatVisible,
          })}
          data-blink={!this.props.chatVisible &&
            messages.length > this.state.readMessages}
          title="Chat"
        >
          <span className="icon icon-question_answer" />
        </div>
        <div
          className="button send-file"
          onClick={this.handleSendFile}
          title="Send file"
        >
          <input
            style={hidden}
            type="file"
            multiple
            ref={this.file}
            onChange={this.handleSelectFiles}
          />
          <span className="icon icon-file-text2" />
        </div>

        {stream && (
          <div>
            <div
              onClick={this.handleMicClick}
              className={classnames('button mute-audio', {
                on: this.state.micMuted,
              })}
              title="Mute audio"
            >
              <span className="on icon icon-mic_off" />
              <span className="off icon icon-mic" />
            </div>
            <div onClick={this.handleCamClick}
              className={classnames('button mute-video', {
                on: this.state.camDisabled,
              })}
              title="Mute video"
            >
              <span className="on icon icon-videocam_off" />
              <span className="off icon icon-videocam" />
            </div>
          </div>
        )}

        <div onClick={this.handleFullscreenClick}
          className={classnames('button fullscreen', {
            on: this.state.fullScreenEnabled,
          })}
          title="Enter fullscreen"
        >
          <span className="on icon icon-fullscreen_exit" />
          <span className="off icon icon-fullscreen" />
        </div>

        <div onClick={this.handleHangoutClick}
          className="button hangup"
          title="Hangup"
        >
          <span className="icon icon-call_end" />
        </div>
      </div>
    )
  }
}
