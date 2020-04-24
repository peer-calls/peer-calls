import React from 'react'
import { StreamWithURL, StreamsState } from '../reducers/streams'
import forEach from 'lodash/forEach'
import { ME } from '../constants'
import { getNickname } from '../nickname'
import Video from './Video'
import { Nicknames } from '../reducers/nicknames'
import { getStreamKey, WindowStates, WindowState } from '../reducers/windowStates'
import { MinimizeTogglePayload } from '../actions/StreamActions'

export interface VideosProps {
  nicknames: Nicknames
  play: () => void
  streams: StreamsState
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  windowStates: WindowStates
}

interface StreamProps {
  key: string
  stream?: StreamWithURL
  userId: string
  muted?: boolean
  localUser?: boolean
  mirrored?: boolean
  windowState: WindowState
}

export default class Videos extends React.PureComponent<VideosProps> {
  private getStreams() {
    const { windowStates, nicknames, streams } = this.props

    const minimized: StreamProps[] = []
    const maximized: StreamProps[] = []

    function addStreamProps(props: StreamProps) {
      if (props.windowState === 'minimized') {
        minimized.push(props)
      } else {
        maximized.push(props)
      }
    }

    function addStreamsByUser(
      localUser: boolean,
      userId: string,
      streams: StreamWithURL[],
    ) {

      if (!streams.length) {
        const key = getStreamKey(userId, undefined)
        const props: StreamProps = {
          key,
          userId,
          localUser,
          windowState: windowStates[key],
        }
        addStreamProps(props)
        return
      }

      streams.forEach((stream, i) => {
        const key = getStreamKey(userId, stream.streamId)
        const props: StreamProps = {
          key,
          stream: stream,
          userId,
          mirrored: localUser && stream.type === 'camera',
          muted: localUser,
          localUser,
          windowState: windowStates[key],
        }
        addStreamProps(props)
      })
    }

    addStreamsByUser(true, ME, streams.localStreams)

    forEach(nicknames, (_, userId) => {
      const s = streams.streamsByUserId[userId]
      addStreamsByUser(false, userId, s && s.streams || [])
    })

    return { minimized, maximized }
  }
  render() {
    const { minimized, maximized } = this.getStreams()

    const videosToolbar = (
      <div className="videos videos-toolbar" key="videos-toolbar">
        {minimized.map(props => (
          <Video
            {...props}
            key={props.key}
            onMinimizeToggle={this.props.onMinimizeToggle}
            play={this.props.play}
            nickname={getNickname(this.props.nicknames, props.userId)}
          />
        ))}
      </div>
    )

    const videosGrid = (
      <div className="videos videos-grid" key="videos-grid">
        {maximized.map(props => (
          <Video
            {...props}
            key={props.key}
            onMinimizeToggle={this.props.onMinimizeToggle}
            play={this.props.play}
            nickname={getNickname(this.props.nicknames, props.userId)}
          />
        ))}
      </div>
    )

    return [videosToolbar, videosGrid]
  }
}
