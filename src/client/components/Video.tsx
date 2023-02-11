import React, { ReactEventHandler } from 'react'
import classnames from 'classnames'
import { StreamWithURL } from '../reducers/streams'
import { Dropdown } from './Dropdown'
import { WindowState } from '../reducers/windowStates'
import { MaximizeParams, MinimizeTogglePayload, StreamDimensionsPayload } from '../actions/StreamActions'
import { MdCrop, MdZoomIn, MdZoomOut, MdMenu } from 'react-icons/md'

import VUMeter from './VUMeter'
import VideoSrc from './VideoSrc'
import { Dim } from '../frame'

export interface VideoProps {
  onMaximize: (payload: MaximizeParams) => void
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  nickname: string
  windowState: WindowState
  stream?: StreamWithURL
  peerId: string
  muted: boolean
  mirrored: boolean
  play: () => void
  localUser?: boolean
  style?: React.CSSProperties
  onDimensions: (payload: StreamDimensionsPayload) => void
  forceContain?: boolean
}

export interface VideoState {
  objectFit: string
}

export default class Video
extends React.PureComponent<VideoProps, VideoState> {
  state = {
    objectFit: '',
  }

  static defaultProps = {
    muted: false,
    mirrored: false,
  }
  handleClick: ReactEventHandler<HTMLVideoElement> = () => {
    this.props.play()
  }
  handleMinimize = () => {
    this.props.onMinimizeToggle({
      peerId: this.props.peerId,
      streamId: this.props.stream && this.props.stream.streamId,
    })
  }
  handleMaximize = () => {
    this.props.onMaximize({
      peerId: this.props.peerId,
      streamId: this.props.stream && this.props.stream.streamId,
    })
  }
  handleToggleCover = () => {
    this.setState({
      objectFit: this.state.objectFit ? '' : 'contain',
    })
  }
  handleLoadedMetadata = (e: React.SyntheticEvent<HTMLVideoElement>) => {
    this.props.play()
  }
  handleResize = (dimensions: Dim) => {
    const { peerId, stream } = this.props
    if (!stream) {
      return
    }

    this.props.onDimensions({
      peerId,
      streamId: stream.streamId,
      dimensions,
    })
  }
  render () {
    const { forceContain, mirrored, peerId, windowState, stream } = this.props
    const minimized =  windowState === 'minimized'
    const className = classnames('video-container', {
      minimized,
      mirrored,
    })

    const streamId = stream && stream.streamId
    const mediaStream = stream && stream.stream || null
    const streamURL = stream && stream.url || ''

    let { objectFit } = this.state

    if (forceContain) {
      objectFit = 'contain'
    }

    return (
      <div className={className} style={this.props.style}>
        <VideoSrc
          id={`video-${peerId}-${streamId}`}
          autoPlay
          onClick={this.handleClick}
          onLoadedMetadata={this.handleLoadedMetadata}
          onResize={this.handleResize}
          muted={this.props.muted}
          mirrored={this.props.mirrored}
          objectFit={objectFit}
          srcObject={mediaStream}
          src={streamURL}
        />
        <div className='video-footer'>
          <VUMeter streamId={streamId} />
          <span className='nickname'>{this.props.nickname}</span>
          <Dropdown fixed label={<MdMenu />}>
            <li className='action-maximize' onClick={this.handleMaximize}>
              <MdZoomIn />&nbsp;
              Maximize
            </li>
            <li className='action-minimize' onClick={this.handleMinimize}>
              {minimized ? <MdZoomIn /> : <MdZoomOut /> }&nbsp;
              Toggle Minimize
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
