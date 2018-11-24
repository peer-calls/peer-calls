import PropTypes from 'prop-types'
import React from 'react'
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
    const document = window.document
    const fs = document.getElementById('container')
    if (
      !document.fullscreenElement &&
      !document.mozFullScreenElement &&
      !document.webkitFullscreenElement &&
      !document.msFullscreenElement
    ) {
      if (fs.requestFullscreen) {
        fs.requestFullscreen()
      } else if (fs.msRequestFullscreen) {
        fs.msRequestFullscreen()
      } else if (fs.mozRequestFullScreen) {
        fs.mozRequestFullScreen()
      } else if (fs.webkitRequestFullscreen) {
        fs.webkitRequestFullscreen()
      }
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen()
      } else if (document.msExitFullscreen) {
        document.msExitFullscreen()
      } else if (document.mozCancelFullScreen) {
        document.mozCancelFullScreen()
      } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen()
      }
    }
    e.currentTarget.classList.toggle('on')
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
          <span className="material-icons">chat</span>
        </div>

        {stream && (
          <div onClick={this.handleMicClick}
            className="button mute-audio"
            title="Mute audio"
          >
            <span className="on material-icons">mic_off</span>
            <span className="off material-icons">mic</span>
          </div>
        )}

        {stream && (
          <div onClick={this.handleCamClick}
            className="button mute-video"
            title="Mute video"
          >
            <span className="on material-icons">videocam_off</span>
            <span className="off material-icons">videocam</span>
          </div>
        )}

        <div onClick={this.handleFullscreenClick}
          className="button fullscreen"
          title="Enter fullscreen"
        >
          <span className="on material-icons">fullscreen_exit</span>
          <span className="off material-icons">fullscreen</span>
        </div>

        <div onClick={this.handleHangoutClick}
          className="button hangup"
          title="Hangup"
        >
          <span className="material-icons">call_end</span>
        </div>
      </div>
    )
  }
}
