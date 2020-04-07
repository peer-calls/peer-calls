import React, { ReactEventHandler } from 'react'
import classnames from 'classnames'
import { StreamWithURL } from '../reducers/streams'
import { NicknameMessage } from '../actions/PeerActions'
import { Nickname } from './Nickname'
import { Dropdown } from './Dropdown'
import { WindowState } from '../reducers/windowStates'
import { MinimizeTogglePayload } from '../actions/StreamActions'

export interface VideoProps {
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  onChangeNickname: (message: NicknameMessage) => void
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
  handleMinimize = () => {
    this.props.onMinimizeToggle({
      userId: this.props.userId,
      streamId: this.props.stream && this.props.stream.stream.id,
    })
  }
  handleToggleCover = () => {
    const v = this.videoRef.current
    if (v) {
      v.style.objectFit = v.style.objectFit ? '' : 'contain'
    }
  }
  render () {
    const { mirrored, muted, userId, windowState } = this.props
    const className = classnames('video-container', {
      minimized: windowState === 'minimized',
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
          muted={muted}
        />
        <div className='video-footer'>
          <Nickname
            value={this.props.nickname}
            onChange={this.props.onChangeNickname}
            localUser={this.props.localUser}
          />
          <Dropdown label={'☰'}>
            <li className='action-minimize' onClick={this.handleMinimize}>
              Toggle Minimize
            </li>
            <li className='action-toggle-fit' onClick={this.handleToggleCover}>
              Toggle Fit
            </li>
          </Dropdown>
        </div>
      </div>
    )
  }
}
