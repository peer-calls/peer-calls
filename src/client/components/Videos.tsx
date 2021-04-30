import React from 'react'
import { connect } from 'react-redux'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { getStreamsByState, StreamProps } from '../selectors'
import { State } from '../store'
import Video from './Video'

export interface VideosProps {
  maximized: StreamProps[]
  minimized: StreamProps[]
  play: () => void
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
}

export class Videos extends React.PureComponent<VideosProps> {
  private gridRef = React.createRef<HTMLDivElement>()
  componentDidUpdate() {
    const videos = this.gridRef.current!
    .querySelectorAll('.video-container') as unknown as HTMLElement[]
    const size = videos.length
    const x = (1 / Math.ceil(Math.sqrt(size))) * 100

    videos.forEach(v => {
      v.style.flexBasis = x + '%'
    })
  }
  render() {
    const { minimized, maximized } = this.props

     const videosToolbar = (
       <div className="videos videos-toolbar" key="videos-toolbar">
         {minimized.map(props => (
           <Video
             {...props}
             key={props.key}
             onMinimizeToggle={this.props.onMinimizeToggle}
             play={this.props.play}
           />
         ))}
       </div>
    )

    const videosGrid = (
      <div className="videos videos-grid" key="videos-grid" ref={this.gridRef}>
        {maximized.map(props => (
          <Video
            {...props}
            key={props.key}
            onMinimizeToggle={this.props.onMinimizeToggle}
            play={this.props.play}
          />
        ))}
      </div>
    )

    return [videosToolbar, videosGrid]
  }
}

function mapStateToProps(state: State) {
  const { minimized, maximized } = getStreamsByState(state)

  return {
    minimized,
    maximized,
  }
}

export default connect(mapStateToProps)(Videos)
