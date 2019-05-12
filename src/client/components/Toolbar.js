import PropTypes from 'prop-types'
import React from 'react'
import screenfull from 'screenfull'
import { MessagePropTypes } from './Chat.js'
import { StreamPropType } from './Video.js'

export default class Toolbar extends React.PureComponent {
  static propTypes = {
    chatRef: PropTypes.object.isRequired,
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired,
    stream: StreamPropType
  }
  constructor () {
    super()
    this.state = {
      isChatOpen: false,
      totalMessages: 0
    }
  }
  handleChatClick = () => {
    const { chatRef, messages } = this.props
    chatRef.classList.toggle('show')
    this.chatButton.classList.toggle('on')
    this.setState({
      isChatOpen: chatRef.classList.contains('show'),
      totalMessages: messages.length
    })
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
    const { isChatOpen, totalMessages } = this.state

    return (
      <div className="toolbar active">
        <div onClick={this.handleChatClick}
          ref={node => { this.chatButton = node }}
          className="button chat"
          data-blink={messages.length !== totalMessages && !isChatOpen}
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
