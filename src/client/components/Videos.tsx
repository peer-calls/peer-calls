import React from 'react'
import { StreamWithURL, StreamsState } from '../reducers/streams'
import forEach from 'lodash/forEach'
import { ME } from '../constants'
import { getNickname } from '../nickname'
import Video from './Video'
import { Nicknames } from '../reducers/nicknames'
import { NicknameMessage } from '../actions/PeerActions'

export interface VideosProps {
  active: string | null
  onChangeNickname: (message: NicknameMessage) => void
  nicknames: Nicknames
  play: () => void
  peers: Record<string, unknown>
  streams: StreamsState
  toggleActive: (userId: string) => void
}

interface StreamProps {
  active: boolean
  key: string
  stream?: StreamWithURL
  userId: string
  muted?: boolean
  localUser?: boolean
  mirrored?: boolean
}

export default class Videos extends React.PureComponent<VideosProps> {
  private getStreams() {
    const { active, peers, streams } = this.props
    let activeProps: StreamProps | undefined
    const otherProps: StreamProps[] = []

    function addStreamsByUser(userId: string) {
      const localUser = userId === ME

      const userStreams = streams[userId]

      if (!userStreams) {
        const key = userId + '_0'
        const isActive = active === key
        const props: StreamProps = {
          active: isActive,
          key,
          userId,
          localUser,
        }
        if (isActive) {
          activeProps = props
        } else {
          otherProps.push(props)
        }
        return
      }

      userStreams.streams.forEach((stream, i) => {
        const key = userId + '_' + i
        const isActive = active === key
        const props: StreamProps = {
          active: isActive,
          key,
          stream: stream,
          userId,
          mirrored: localUser && stream.type === 'camera',
          muted: localUser,
          localUser,
        }
        if (isActive) {
          activeProps = props
        } else {
          otherProps.push(props)
        }
      })
    }

    addStreamsByUser(ME)
    forEach(peers, (_, userId) => addStreamsByUser(userId))

    return { activeProps, otherProps }
  }
  render() {
    const { activeProps, otherProps } = this.getStreams()

    const activeVideo = activeProps && (
      <Video
        {...activeProps}
        key={activeProps.key}
        userId={activeProps.key}
        onClick={this.props.toggleActive}
        play={this.props.play}
        nickname={getNickname(this.props.nicknames, activeProps.userId)}
        onChangeNickname={this.props.onChangeNickname}
      />
    )

    const otherVideos = (
      <div className="videos" key="videos">
        {otherProps.map(props => (
          <Video
            {...props}
            userId={props.key}
            key={props.key}
            onClick={this.props.toggleActive}
            play={this.props.play}
            nickname={getNickname(this.props.nicknames, props.userId)}
            onChangeNickname={this.props.onChangeNickname}
          />
        ))}
      </div>
    )
    return [activeVideo, otherVideos]
  }
}
