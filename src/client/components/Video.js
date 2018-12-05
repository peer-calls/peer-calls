import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import { MediaStream } from '../window.js'
import socket from '../socket.js'

export const StreamPropType = PropTypes.shape({
  mediaStream: PropTypes.instanceOf(MediaStream).isRequired,
  url: PropTypes.string
})

export default class Video extends React.PureComponent {
  static propTypes = {
    videos: PropTypes.object.isRequired,
    onClick: PropTypes.func,
    active: PropTypes.bool.isRequired,
    stream: StreamPropType,
    userId: PropTypes.string.isRequired
  }
  handleClick = e => {
    const { onClick, userId } = this.props
    this.play(e)
    onClick(userId)
  }
  play = e => {
    e.preventDefault()
    e.target.play()
  }
  componentDidMount () {
    this.componentDidUpdate()
  }
  componentDidUpdate () {
    const { videos, stream } = this.props
    const { video } = this.refs
    const mediaStream = stream && stream.mediaStream
    const url = stream && stream.url
    if ('srcObject' in video) {
      if (video.srcObject !== mediaStream) {
        this.refs.video.srcObject = mediaStream
      }
    } else if (video.src !== url) {
      video.src = url
    }
    if (socket.id) {
      videos[socket.id] = video
    }
  }
  render () {
    const { active } = this.props
    const className = classnames('video-container', { active })
    return (
      <div className={className}>
        <video
          id={`video-${socket.id}`}
          autoPlay
          onClick={this.handleClick}
          onLoadedMetadata={this.play}
          playsInline
          ref="video"
        />
      </div>
    )
  }
}
