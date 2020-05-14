import React, { ReactEventHandler } from 'react'
import classnames from 'classnames'
import { StreamWithURL } from '../reducers/streams'
import { Dropdown } from './Dropdown'
import { WindowState } from '../reducers/windowStates'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { MdCrop, MdZoomIn, MdZoomOut, MdMenu } from 'react-icons/md'

export interface VideoProps {
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  nickname: string
  windowState: WindowState
  stream?: StreamWithURL
  userId: string
  muted: boolean
  mirrored: boolean
  play: () => void
  localUser?: boolean
}

export default class Video extends React.PureComponent<VideoProps> {
  videoRef = React.createRef<HTMLVideoElement>()

  static defaultProps = {
    muted: false,
    mirrored: false,
  }
  handleClick: ReactEventHandler<HTMLVideoElement> = e => {
    this.props.play()
  }
  componentDidMount () {
    this.componentDidUpdate()
  }
  componentDidUpdate () {
    const { stream } = this.props
    const video = this.videoRef.current
    if (video) {
      const mediaStream = stream && stream.stream || null
      const url = stream && stream.url
      if ('srcObject' in video as unknown) {
        if (video.srcObject !== mediaStream) {
          video.srcObject = mediaStream
        }
      } else if (video.src !== url) {
        video.src = url || ''
      }
      video.muted = this.props.muted
    }
  }
  handleMinimize = () => {
    this.props.onMinimizeToggle({
      userId: this.props.userId,
      streamId: this.props.stream && this.props.stream.streamId,
    })
  }
  handleToggleCover = () => {
    const v = this.videoRef.current
    if (v) {
      v.style.objectFit = v.style.objectFit ? '' : 'contain'
    }
  }
  render () {
    const { mirrored, userId, windowState } = this.props
    const minimized =  windowState === 'minimized'
    const className = classnames('video-container', {
      minimized,
      mirrored,
    })

    return (
      <div className={className}>
        <video
          id={`video-${userId}`}
          autoPlay
          onClick={this.handleClick}
          onLoadedMetadata={() => this.props.play()}
          playsInline
          ref={this.videoRef}
        />
        <div className='video-footer'>
          <span className='nickname'>{this.props.nickname}</span>
          <Dropdown label={<MdMenu />}>
            <li className='action-minimize' onClick={this.handleMinimize}>
              {minimized ? <MdZoomIn /> : <MdZoomOut /> }&nbsp;
              {minimized ? 'Maximize': 'Minimize' }
            </li>
            <li className='action-toggle-fit' onClick={this.handleToggleCover}>
              <MdCrop /> Toggle Fit
            </li>
          </Dropdown>
        </div>
      </div>
    )
  }
}
