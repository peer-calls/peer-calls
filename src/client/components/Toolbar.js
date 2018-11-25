import PropTypes from 'prop-types'
import React from 'react'
import screenfull from 'screenfull'
import { MessagePropTypes } from './Chat.js'
import { StreamPropType } from './Video.js'

export default class Toolbar extends React.PureComponent {
  static propTypes = {
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
  handleChatClick = e => {
    const { messages } = this.props
    document.getElementById('chat').classList.toggle('show')
    e.currentTarget.classList.toggle('on')
    this.setState({
      isChatOpen: document.getElementById('chat').classList.contains('show'),
      totalMessages: messages.length
    })
  }
  handleMicClick = e => {
    const { stream } = this.props
    stream.mediaStream.getAudioTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    e.currentTarget.classList.toggle('on')
  }
  handleCamClick = e => {
    const { stream } = this.props
    stream.mediaStream.getVideoTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    e.currentTarget.classList.toggle('on')
  }
  handleFullscreenClick = e => {
    if (screenfull.enabled) {
      screenfull.toggle(e.target)
      e.currentTarget.classList.toggle('on')
    }
  }
  handleHangoutClick = e => {
    window.location.href = '/'
  }
  render () {
    const { messages, stream } = this.props
    const { isChatOpen, totalMessages } = this.state

    return (
      <div className="toolbar active">
        <div onClick={this.handleChatClick}
          className="button chat"
          data-blink={messages.length !== totalMessages && !isChatOpen}
          title="Chat"
        >
          <span className="icon icon-question_answer" />
        </div>

        {stream && (
          <div onClick={this.handleMicClick}
            className="button mute-audio"
            title="Mute audio"
          >
            <span className="on icon icon-mic_off" />
            <span className="off icon icon-mic" />
          </div>
        )}

        {stream && (
          <div onClick={this.handleCamClick}
            className="button mute-video"
            title="Mute video"
          >
            <span className="on icon icon-videocam_off" />
            <span className="off icon icon-videocam" />
          </div>
        )}

        <div onClick={this.handleFullscreenClick}
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
