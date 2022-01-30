import classNames from 'classnames'
import React from 'react'
import { connect } from 'react-redux'
import ResizeObserver from 'resize-observer-polyfill'
import { GridKind } from '../actions/SettingsActions'
import { MaximizeParams, MinimizeTogglePayload } from '../actions/StreamActions'
import { SETTINGS_GRID_ASPECT, SETTINGS_GRID_AUTO } from '../constants'
import { Dim, Frame } from '../frame'
import { getStreamsByState, StreamProps } from '../selectors'
import { State } from '../store'
import Video from './Video'

export interface VideosProps {
  maximized: StreamProps[]
  minimized: StreamProps[]
  play: () => void
  onMaximize: (payload: MaximizeParams) => void
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  showMinimizedToolbar: boolean
  gridKind: GridKind
  defaultAspectRatio: number
  debug: boolean
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
      return defaultAspectRatio
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
  render() {
    const {
      minimized,
      maximized,
      showMinimizedToolbar,
    } = this.props

    const aspectRatio = this.getAspectRatio()

    const windows = maximized

    this.maybeUpdateSizeStyle()

    const toolbarClassName = classNames('videos videos-toolbar', {
      'hidden': !showMinimizedToolbar || minimized.length === 0,
    })

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
            onMaximize={this.props.onMaximize}
            onMinimizeToggle={this.props.onMinimizeToggle}
            play={this.props.play}
            style={this.state.toolbarVideoStyle}
          />
        ))}
      </div>
    )

    const isAspectRatio = aspectRatio > 0

    const maximizedVideos = windows.map(props => (
      <Video
        {...props}
        key={props.key}
        onMaximize={this.props.onMaximize}
        onMinimizeToggle={this.props.onMinimizeToggle}
        play={this.props.play}
        style={this.videoStyle}
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

function mapStateToProps(state: State) {
  const { minimized, maximized } = getStreamsByState(state)
  const { gridKind } = state.settings

  return {
    minimized,
    maximized,
    gridKind,
    defaultAspectRatio: 16/9,
    debug: true,
  }
}

export default connect(mapStateToProps)(Videos)
