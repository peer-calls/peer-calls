import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import screenfull from 'screenfull'
import { MessagePropTypes } from './Chat.js'
import { StreamPropType } from './Video.js'

export default class Toolbar extends React.PureComponent {
  static propTypes = {
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired,
    stream: StreamPropType,
    onToggleChat: PropTypes.func.isRequired,
    chatVisible: PropTypes.bool.isRequired
  }
  handleMicClick = () => {
    const { stream } = this.props
    stream.mediaStream.getAudioTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    this.mixButton.classList.toggle('on')
  }
  handleCamClick = () => {
    const { stream } = this.props
    stream.mediaStream.getVideoTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    this.camButton.classList.toggle('on')
  }
  handleFullscreenClick = () => {
    if (screenfull.enabled) {
      screenfull.toggle()
      this.fullscreenButton.classList.toggle('on')
    }
  }
  handleHangoutClick = () => {
    window.location.href = '/'
  }
  render () {
    const { messages, stream } = this.props

    return (
      <div className="toolbar active">
        <div onClick={this.props.onToggleChat}
          className={classnames('button chat', {
            on: this.props.chatVisible
          })}
          data-blink={this.props.chatVisible && messages.length}
          title="Chat"
        >
          <span className="icon icon-question_answer" />
        </div>

        {stream && (
          <div>
            <div onClick={this.handleMicClick}
              ref={node => { this.mixButton = node }}
              className="button mute-audio"
              title="Mute audio"
            >
              <span className="on icon icon-mic_off" />
              <span className="off icon icon-mic" />
            </div>
            <div onClick={this.handleCamClick}
              ref={node => { this.camButton = node }}
              className="button mute-video"
              title="Mute video"
            >
              <span className="on icon icon-videocam_off" />
              <span className="off icon icon-videocam" />
            </div>
          </div>
        )}

        <div onClick={this.handleFullscreenClick}
          ref={node => { this.fullscreenButton = node }}
          className="button fullscreen"
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
