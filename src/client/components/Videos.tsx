import React from 'react'
import { StreamWithURL, StreamsState } from '../reducers/streams'
import forEach from 'lodash/forEach'
import { ME } from '../constants'
import { getNickname } from '../nickname'
import Video from './Video'
import { Nicknames } from '../reducers/nicknames'
import { NicknameMessage } from '../actions/PeerActions'
import { getStreamKey, WindowStates, WindowState } from '../reducers/windowStates'
import { MinimizeTogglePayload } from '../actions/StreamActions'

export interface VideosProps {
  onChangeNickname: (message: NicknameMessage) => void
  nicknames: Nicknames
  play: () => void
  peers: Record<string, unknown>
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
    const { windowStates, peers, streams } = this.props

    const minimized: StreamProps[] = []
    const maximized: StreamProps[] = []

    function addStreamProps(props: StreamProps) {
      if (props.windowState === 'minimized') {
        minimized.push(props)
      } else {
        maximized.push(props)
      }
    }

    function addStreamsByUser(userId: string) {
      const localUser = userId === ME

      const userStreams = streams[userId]

      if (!userStreams) {
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

      userStreams.streams.forEach((stream, i) => {
        const key = getStreamKey(userId, stream.stream.id)
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

    addStreamsByUser(ME)
    forEach(peers, (_, userId) => addStreamsByUser(userId))

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
            onChangeNickname={this.props.onChangeNickname}
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
            onChangeNickname={this.props.onChangeNickname}
          />
        ))}
      </div>
    )

    return [videosToolbar, videosGrid]
  }
}
