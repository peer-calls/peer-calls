import React from 'react'
import { StreamPropType } from './video.js'

export default class Toolbar extends React.PureComponent {
  static propTypes = {
    stream: StreamPropType
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
    const { stream } = this.props

    return (
      <div className="toolbar active">

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
