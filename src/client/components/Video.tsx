import classnames from 'classnames'
import React, { ReactEventHandler } from 'react'
import { MdCrop, MdInfoOutline, MdMenu, MdZoomIn, MdZoomOut } from 'react-icons/md'
import { MaximizeParams, MinimizeTogglePayload, StreamDimensionsPayload } from '../actions/StreamActions'
import { Dim } from '../frame'
import { ReceiverStatsParams } from '../reducers/receivers'
import { StreamWithURL } from '../reducers/streams'
import { WindowState } from '../reducers/windowStates'
import { Dropdown } from './Dropdown'
import Stats from './Stats'
import VideoSrc from './VideoSrc'
import VUMeter from './VUMeter'

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
  getReceiverStats: (
    params: ReceiverStatsParams,
  ) => Promise<RTCStatsReport | undefined>
  getSenderStats: (
    track: MediaStreamTrack,
  ) => Promise<{peerId: string, stats: RTCStatsReport}[]>
  showStats?: boolean
}

export interface VideoState {
  objectFit: string
  showStats: boolean
}

export default class Video
extends React.PureComponent<VideoProps, VideoState> {
  state = {
    objectFit: '',
    showStats: false,
  }

  statsTimeout: NodeJS.Timeout | undefined

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
  handleToggleStats = () => {
    this.setState({
      showStats: !this.state.showStats,
    })
  }
  render () {
    const { forceContain, mirrored, peerId, windowState, stream } = this.props
    const showStats = this.state.showStats || this.props.showStats
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
        {showStats && (
          <div className='video-stats'>
            <Stats
              stream={stream}
              peerId={this.props.peerId}
              getReceiverStats={this.props.getReceiverStats}
              getSenderStats={this.props.getSenderStats}
            />
          </div>
        )}
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
            {!forceContain && (
              <li
              className='action-toggle-fit'
              onClick={this.handleToggleCover}
              >
                <MdCrop /> Toggle Fit
              </li>
            )}
            {stream && !this.props.showStats && (<li
              className='action-toggle-stats' onClick={this.handleToggleStats}
            >
              <MdInfoOutline /> Stats
            </li>)}
          </Dropdown>
        </div>
      </div>
    )
  }
}
