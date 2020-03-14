import React, { ReactEventHandler } from 'react'
import classnames from 'classnames'
import { StreamWithURL } from '../reducers/streams'
import { NicknameMessage } from '../actions/PeerActions'
import { Nickname } from './Nickname'

export interface VideoProps {
  // videos: Record<string, unknown>
  onClick: (userId: string) => void
  onChangeNickname: (message: NicknameMessage) => void
  nickname: string
  active: boolean
  stream?: StreamWithURL
  userId: string
  muted: boolean
  mirrored: boolean
  play: () => void
  localUser?: boolean
}

export default class Video extends React.PureComponent<VideoProps> {
  videoRef = React.createRef<HTMLVideoElement>()
  timeout?: number

  static defaultProps = {
    muted: false,
    mirrored: false,
  }
  handleClick: ReactEventHandler<HTMLVideoElement> = e => {
    const { onClick, userId } = this.props
    if (this.timeout) {
      // if the timeout was cancelled, execute click
      this.props.play()
      onClick(userId)
    }
    this.timeout = undefined
  }
  handleMouseDown: ReactEventHandler<HTMLVideoElement> = e => {
    clearTimeout(this.timeout)
    this.timeout = window.setTimeout(this.toggleCover, 300)
  }
  handleMouseUp: ReactEventHandler<HTMLVideoElement> = e => {
    clearTimeout(this.timeout)
  }
  toggleCover = () => {
    this.timeout = undefined
    const v = this.videoRef.current
    if (v) {
      v.style.objectFit = v.style.objectFit ? '' : 'cover'
    }
  }
  componentDidMount () {
    this.componentDidUpdate()
  }
  componentDidUpdate () {
    const { stream } = this.props
    const video = this.videoRef.current!
    const mediaStream = stream && stream.stream || null
    const url = stream && stream.url
    if ('srcObject' in video as unknown) {
      if (video.srcObject !== mediaStream) {
        video.srcObject = mediaStream
      }
    } else if (video.src !== url) {
      video.src = url || ''
    }
  }
  render () {
    const { active, mirrored, muted, userId } = this.props
    const className = classnames('video-container', { active, mirrored })
    return (
      <div className={className}>
        <video
          id={`video-${userId}`}
          autoPlay
          onClick={this.handleClick}
          onMouseDown={this.handleMouseDown}
          onTouchStart={this.handleMouseDown}
          onMouseUp={this.handleMouseUp}
          onTouchEnd={this.handleMouseUp}
          onLoadedMetadata={() => this.props.play()}
          playsInline
          ref={this.videoRef}
          muted={muted}
        />
        <Nickname
          value={this.props.nickname}
          onChange={this.props.onChangeNickname}
          localUser={this.props.localUser}
        />
      </div>
    )
  }
}
