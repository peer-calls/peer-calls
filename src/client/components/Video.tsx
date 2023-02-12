import classnames from 'classnames'
import React, { ReactEventHandler } from 'react'
import { MdCrop, MdInfoOutline, MdMenu, MdZoomIn, MdZoomOut } from 'react-icons/md'
import { MaximizeParams, MinimizeTogglePayload, StreamDimensionsPayload } from '../actions/StreamActions'
import { ME } from '../constants'
import { Dim } from '../frame'
import { ReceiverStatsParams } from '../reducers/receivers'
import { StreamWithURL } from '../reducers/streams'
import { WindowState } from '../reducers/windowStates'
import { Dropdown } from './Dropdown'
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
}

export interface VideoState {
  objectFit: string
  showStats: boolean
  statsReport: string
}

export default class Video
extends React.PureComponent<VideoProps, VideoState> {
  state = {
    objectFit: '',
    showStats: false,
    statsReport: '',
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
    const { peerId } = this.props

    const showStats = !this.state.showStats

    this.setState({
      showStats,
    })

    if (this.statsTimeout) {
      clearTimeout(this.statsTimeout)
      this.statsTimeout = undefined
    }

    if (!showStats) {
      return
    }

    this.statsTimeout = setInterval(async () => {
      let statsReport = ''
      if (peerId === ME) {
        statsReport = await this.fetchSenderStats()
      } else {
        statsReport = await this.fetchReceiverStats()
      }

      this.setState({
        statsReport,
      })
    }, 1000)
  }
  buildStatsReport = (stats: RTCStatsReport, sections: string[]): string => {
    let r = ''

    const set = new Set(sections)

    stats.forEach(v => {
      if (!set.has(v.type)) {
        return
      }

      const i = v as RTCInboundRTPStreamStats
      const o = v as RTCOutboundRTPStreamStats

      switch (v.type) {
      case 'codec':
        r += 'Channels: ' + v.channels + '\n'
        r += 'Clock rate: ' + v.clockRate + '\n'
        r += 'MIME Type: ' + v.mimeType + '\n'
        r += 'Payload Type: ' + v.payloadType + '\n'
        r += 'SDP FMTP Line: ' + v.sdpFmtpLine + '\n'
        break
      case 'inbound-rtp':
        r += 'SSRC: ' + i.ssrc + '\n'
        r += 'Bytes received: ' + i.bytesReceived + '\n'
        r += 'Packets received: ' + i.packetsReceived + '\n'
        r += 'Packets discarded: ' + v.packetsDiscarded + '\n'
        r += 'Packets lost: ' + i.packetsLost + '\n'
        r += 'FIR count: ' + i.firCount + '\n'
        r += 'PLI count: ' + i.pliCount + '\n'
        r += 'NACK count: ' + i.nackCount + '\n'
        r += 'SLI count: ' + i.sliCount + '\n'
        break
      case 'outbound-rtp':
        r += 'SSRC: ' + o.ssrc + '\n'
        r += 'Bytes sent: ' + o.bytesSent + '\n'
        r += 'Packets sent: ' + o.packetsSent + '\n'
        r += 'FIR count: ' + o.firCount + '\n'
        r += 'PLI count: ' + o.pliCount + '\n'
        r += 'NACK count: ' + o.nackCount + '\n'
        r += 'SLI count: ' + o.sliCount + '\n'
        r += 'Round trip time: ' + o.roundTripTime + '\n'
      break
      default:
          // Do nothing.
      }
    })

    return r
  }
  fetchReceiverStats = async () => {
    const { stream, getReceiverStats } = this.props

    if (!stream) {
      return 'No stream'
    }

    const streamId = stream.streamId

    const tracks = stream.stream.getTracks()

    const tps = tracks.map(track => {
      return {
        track,
        promise: getReceiverStats({
          streamId,
          trackId: track.id,
        }),
      }
    })

    const reports: string[] = []
    const sections = ['codec', 'inbound-rtp']

    for (const tp of tps) {
      const { track, promise } = tp

      let r = ''

      r += `${track.kind.toUpperCase()} ${track.id}\n`

      try {
        const stats = await promise
        if (stats) {
          r += this.buildStatsReport(stats, sections)
        } else {
          r += 'No report available\n'
        }
      } catch (err) {
        r += 'Error ' + err + '\n'
      }

      reports.push(r)
    }

    return reports.join('\n')
  }
  fetchSenderStats = async () => {
    const { stream, getSenderStats } = this.props

    if (!stream) {
      return 'No stream'
    }

    const tracks = stream.stream.getTracks()

    const tps = tracks.map(track => {
      return {
        track,
        promise: getSenderStats(track),
      }
    })

    const reports: string[] = []
    const sections = ['codec', 'outbound-rtp']

    for (const tp of tps) {
      const { track, promise } = tp

      let r = `${track.kind.toUpperCase()} ${track.id}\n`
      let statsPerPeer = []

      try {
        statsPerPeer = await promise
      } catch (err) {
        r += 'Error ' + err + '\n'
        continue
      }

      if (!statsPerPeer.length) {
        r += 'No report available\n'
        continue
      }

      statsPerPeer.forEach(s => {
        const { peerId, stats } = s

        r += `Peer ID: ${peerId}` + '\n'
        r += this.buildStatsReport(stats, sections)
      })

      reports.push(r)
    }

    return reports.join('\n')
  }
  render () {
    const { forceContain, mirrored, peerId, windowState, stream } = this.props
    const { showStats } = this.state
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
            {this.state.statsReport}
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
            <li className='action-toggle-fit' onClick={this.handleToggleCover}>
              <MdCrop /> Toggle Fit
            </li>
            {stream && (<li
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
