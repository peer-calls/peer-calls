import PropTypes from 'prop-types'
import React from 'react'
import screenfull from 'screenfull'
import { MessagePropTypes } from './Chat.js'
import { StreamPropType } from './Video.js'

export default class Toolbar extends React.PureComponent {
  static propTypes = {
    drawerRef: PropTypes.object.isRequired,
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired,
    stream: StreamPropType
  }
  constructor () {
    super()
    this.state = {
      isDrawerOpen: false,
      totalMessages: 0
    }
  }
  handleDrawerClick = () => {
    const { drawerRef, messages } = this.props
    drawerRef.classList.toggle('show')
    this.drawerButton.classList.toggle('on')
    this.setState({
      isDrawerOpen: drawerRef.classList.contains('show'),
      totalMessages: messages.length
    })
  }
  handleMicClick = () => {
    const { stream } = this.props
    stream.mediaStream.getAudioTracks().forEach(track => {
      track.enabled = !track.enabled
    })
    this.micButton.classList.toggle('on')
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
      screenfull.toggle(this.fullscreenButton)
      this.fullscreenButton.classList.toggle('on')
    }
  }
  handleHangoutClick = () => {
    window.location.href = '/'
  }
  render () {
    const { messages, stream } = this.props
    const { isDrawerOpen, totalMessages } = this.state

    return (
      <div className="toolbar active">
        <div onClick={this.handleDrawerClick}
          ref={node => { this.drawerButton = node }}
          className="button drawer"
          data-blink={messages.length !== totalMessages && !isDrawerOpen}
          title="Drawer"
        >
          <span className="icon icon-question_answer" />
        </div>

        {stream && (
          <div>
            <div onClick={this.handleMicClick}
              ref={node => { this.micButton = node }}
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
