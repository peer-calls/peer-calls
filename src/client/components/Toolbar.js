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
    e.target.classList.toggle('on')
  }
  handleHangoutClick = e => {
    window.location.href = '/'
  }
  render () {
    const { stream } = this.props

    return (
      <div className="toolbar active">

        {stream && (
          <svg onClick={this.handleMicClick} className="mute-audio" xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 1024 1024">
            <circle cx="24" cy="24" r="34">
              <title>Mute audio</title>
            </circle>
            <path className="on" transform="scale(0.4), translate(800,800)" d="M182 149.333l714 714-54 54-178-178c-32 20-72 32-110 38v140h-84v-140c-140-20-256-140-256-286h72c0 128 108 216 226 216 34 0 68-8 98-22l-70-70c-8 2-18 4-28 4-70 0-128-58-128-128v-32l-256-256zm458 348l-256-254v-8c0-70 58-128 128-128s128 58 128 128v262zm170-6c0 50-14 98-38 140l-52-54c12-26 18-54 18-86h72z" fill="white" />
            <path className="off" transform="scale(0.4), translate(800,800)" d="M738 491.333h72c0 146-116 266-256 286v140h-84v-140c-140-20-256-140-256-286h72c0 128 108 216 226 216s226-88 226-216zm-226 128c-70 0-128-58-128-128v-256c0-70 58-128 128-128s128 58 128 128v256c0 70-58 128-128 128z" fill="white" />
          </svg>
        )}

        {stream && (
          <svg onClick={this.handleCamClick} className="mute-video" xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 1024 1024">
            <circle cx="24" cy="24" r="34">
              <title>Mute video</title>
            </circle>
            <path className="on" transform="scale(0.4), translate(800,800)" d="M140 107.333l756 756-54 54-136-136c-6 4-16 8-24 8H170c-24 0-42-18-42-42v-428c0-24 18-42 42-42h32l-116-116zm756 192v456l-478-478h264c24 0 44 18 44 42v150z" fill="white" />
            <path className="off" transform="scale(0.4), translate(800,800)" d="M726 469.333l170-170v468l-170-170v150c0 24-20 42-44 42H170c-24 0-42-18-42-42v-428c0-24 18-42 42-42h512c24 0 44 18 44 42v150z" fill="white" />
          </svg>
        )}

        <svg onClick={this.handleFullscreenClick} className="fullscreen" xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 1024 1024">
          <circle cx="24" cy="24" r="34">
            <title>Enter fullscreen</title>
          </circle>
          <path className="on" transform="scale(0.4), translate(800,800)" d="M682 363.333h128v84H598v-212h84v128zm-84 468v-212h212v84H682v128h-84zm-256-468v-128h84v212H214v-84h128zm-128 340v-84h212v212h-84v-128H214z" fill="white" />
          <path className="off" transform="scale(0.4), translate(800,800)" d="M598 235.333h212v212h-84v-128H598v-84zm128 512v-128h84v212H598v-84h128zm-512-300v-212h212v84H298v128h-84zm84 172v128h128v84H214v-212h84z" fill="white" />
        </svg>

        <svg onClick={this.handleHangoutClick} className="hangup" xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 1024 1024">
          <circle cx="24" cy="24" r="34">
            <title>Hangup</title>
          </circle>
          <path transform="scale(0.4), translate(800,800)" d="M512 405.333c-68 0-134 10-196 30v132c0 16-10 34-24 40-42 20-80 46-114 78-8 8-18 12-30 12s-22-4-30-12l-106-106c-8-8-12-18-12-30s4-22 12-30c130-124 306-200 500-200s370 76 500 200c8 8 12 18 12 30s-4 22-12 30l-106 106c-8 8-18 12-30 12s-22-4-30-12c-34-32-72-58-114-78-14-6-24-20-24-38v-132c-62-20-128-32-196-32z" fill="white" />
        </svg>
      </div>
    )
  }
}
