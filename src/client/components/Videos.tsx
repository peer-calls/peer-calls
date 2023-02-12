import classNames from 'classnames'
import map from 'lodash/map'
import React from 'react'
import { connect } from 'react-redux'
import ResizeObserver from 'resize-observer-polyfill'
import { GridKind } from '../actions/SettingsActions'
import { MaximizeParams, MinimizeTogglePayload, StreamDimensionsPayload } from '../actions/StreamActions'
import { SETTINGS_GRID_ASPECT, SETTINGS_GRID_AUTO } from '../constants'
import { Dim, Frame } from '../frame'
import { PeersState } from '../reducers/peers'
import { ReceiversState } from '../reducers/receivers'
import { createReceiverStatsKey, ReceiverStatsParams } from '../reducers/receivers'
import { getStreamsByState, StreamProps } from '../selectors'
import { State } from '../store'
import Video from './Video'

export interface VideosProps {
  maximized: StreamProps[]
  minimized: StreamProps[]
  play: () => void
  onMaximize: (payload: MaximizeParams) => void
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  onDimensions: (payload: StreamDimensionsPayload) => void
  showMinimizedToolbar: boolean
  gridKind: GridKind
  defaultAspectRatio: number
  debug: boolean
  receivers: ReceiversState
  peers: PeersState
  showAllStats: boolean
}

export interface VideosState {
  videoSize: Dim
  toolbarVideoStyle: React.CSSProperties
}

export class Videos extends React.PureComponent<VideosProps, VideosState> {
  private gridRef = React.createRef<HTMLDivElement>()
  private toolbarRef = React.createRef<HTMLDivElement>()
  private frame: Frame
  private videoStyle?: React.CSSProperties
  private gridObserver: ResizeObserver
  private toolbarObserver: ResizeObserver

  constructor(props: VideosProps) {
    super(props)

    this.state = {
      videoSize: {x: 0, y: 0},
      toolbarVideoStyle: {},
    }

    this.frame = new Frame(this.props.defaultAspectRatio)

    this.gridObserver = new ResizeObserver(this.handleResize)
    this.toolbarObserver = new ResizeObserver(this.handleToolbarResize)
  }
  getAspectRatio = (): number => {
    const { defaultAspectRatio, gridKind, maximized } = this.props

    const numWindows = maximized.length

    if (
      gridKind === SETTINGS_GRID_ASPECT ||
      gridKind === SETTINGS_GRID_AUTO && numWindows > 2
    ) {
      return calcAspectRatio(defaultAspectRatio, maximized)
    }

    return 0
  }
  componentDidMount = () => {
    this.handleResize()
    this.handleToolbarResize()

    this.gridObserver.observe(this.gridRef.current!)
    this.toolbarObserver.observe(this.toolbarRef.current!)
  }
  componentWillUnmount = () => {
    this.gridObserver.disconnect()
    this.toolbarObserver.disconnect()
  }
  handleToolbarResize = () => {
    const size = getSize(this.toolbarRef)

    const aspectRatio = this.props.defaultAspectRatio

    this.setState({
      toolbarVideoStyle: {
        width: Math.round(size.y * aspectRatio * 100) / 100,
        height: size.y,
      },
    })
  }
  handleResize = () => {
    const size = getSize(this.gridRef)

    this.frame.setSize(size)

    this.setState({
      videoSize: size,
    })
  }
  componentDidUpdate() {
    if (this.getAspectRatio()) {
      return
    }

    const videos = this.gridRef.current!
    .querySelectorAll('.video-container') as unknown as HTMLElement[]
    const size = videos.length
    const x = (1 / Math.ceil(Math.sqrt(size))) * 100

    videos.forEach(v => {
      v.style.flexBasis = x + '%'
    })
  }
  maybeUpdateSizeStyle() {
    const {maximized} = this.props

    const aspectRatio = this.getAspectRatio()

    if (!aspectRatio) {
      this.videoStyle = undefined
      return
    }

    this.frame.setAspectRatio(aspectRatio)
    this.frame.setNumWindows(maximized.length)

    if (this.frame.needsCalc() || !this.videoStyle) {
      const size = this.frame.calcSize()

      this.videoStyle = {
        width: size.x,
        height: size.y,
      }
    }
  }
  getReceiverStats = async (params: ReceiverStatsParams) => {
    const key = createReceiverStatsKey(params)
    const receiver = this.props.receivers[key]
    if (!receiver) {
      return
    }

    return await receiver.getStats()
  }
  getSenderStats = async (track: MediaStreamTrack) => {
    const rp = map(this.props.peers, (peer, peerId) => {
      const sender = peer.senders[track.id]

      if (!sender) {
        return
      }

      return {
        peerId,
        promise: sender.getStats(),
      }
    })

    const ret = []

    for (const report of rp) {
      if (!report) {
        continue
      }

      const stats = await report.promise

      ret.push({
        peerId: report.peerId,
        stats,
      })
    }

    return ret
  }
  render() {
    const {
      minimized,
      maximized,
      showMinimizedToolbar,
    } = this.props

    const windows = maximized

    this.maybeUpdateSizeStyle()

    const toolbarClassName = classNames('videos videos-toolbar', {
      'hidden': !showMinimizedToolbar || minimized.length === 0,
    })

    const isAspectRatio = this.videoStyle !== undefined

    const videosToolbar = (
      <div
        className={toolbarClassName}
        key="videos-toolbar"
        ref={this.toolbarRef}
      >
        {minimized.map(props => (
          <Video
            {...props}
            key={props.key}
            onDimensions={this.props.onDimensions}
            onMaximize={this.props.onMaximize}
            onMinimizeToggle={this.props.onMinimizeToggle}
            play={this.props.play}
            style={this.state.toolbarVideoStyle}
            forceContain={isAspectRatio}
            getReceiverStats={this.getReceiverStats}
            getSenderStats={this.getSenderStats}
            showStats={this.props.showAllStats}
          />
        ))}
      </div>
    )

    const maximizedVideos = windows.map(props => (
      <Video
        {...props}
        key={props.key}
        onDimensions={this.props.onDimensions}
        onMaximize={this.props.onMaximize}
        onMinimizeToggle={this.props.onMinimizeToggle}
        play={this.props.play}
        style={this.videoStyle}
        forceContain={isAspectRatio}
        getReceiverStats={this.getReceiverStats}
        getSenderStats={this.getSenderStats}
        showStats={this.props.showAllStats}
      />
    ))

    const videosGrid = isAspectRatio
    ? (
      <div
        className='videos-grid videos-grid-aspect-ratio'
        key='videos-grid'
        ref={this.gridRef}
      >
        <div className='videos-grid-aspect-ratio-container'>
          {maximizedVideos}
        </div>
      </div>
    )
    : (
      <div
        className='videos-grid videos-grid-flex'
        key='videos-grid'
        ref={this.gridRef}
      >
        {maximizedVideos}
      </div>
    )

    return [videosToolbar, videosGrid]
  }
}

function getSize<T extends HTMLElement>(ref: React.RefObject<T>): Dim {
  const {width: x, height: y} = ref.current!.getBoundingClientRect()
  return {x, y}
}

function calcAspectRatio(
  defaultAspectRatio: number,
  streamProps: StreamProps[],
): number {
  let ratio = 0

  for (let i = 0; i < streamProps.length; i++) {
    const stream = streamProps[i].stream
    if (!stream) {
      continue
    }

    const dim = stream.dimensions
    if (!dim) {
      continue
    }

    const r = dim.x / dim.y

    if (ratio === 0) {
      ratio = r
      continue
    }

    if (ratio !== r) {
      ratio = defaultAspectRatio
      break
    }
  }

  return ratio || defaultAspectRatio
}

function mapStateToProps(state: State) {
  const { minimized, maximized } = getStreamsByState(state)
  const { gridKind, showAllStats } = state.settings

  return {
    minimized,
    maximized,
    gridKind,
    defaultAspectRatio: 16/9,
    debug: true,
    receivers: state.receivers,
    peers: state.peers,
    showAllStats,
  }
}

export default connect(mapStateToProps)(Videos)
