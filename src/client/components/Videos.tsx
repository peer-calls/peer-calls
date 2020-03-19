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
  streams: StreamsState
  toggleActive: (userId: string) => void
}

interface StreamProps {
  active: boolean
  key: string
  stream: StreamWithURL
  userId: string
  muted?: boolean
  localUser?: boolean
  mirrored?: boolean
}

export default class Videos extends React.PureComponent<VideosProps> {
  private getStreams() {
    const { active } = this.props
    let activeProps: StreamProps | undefined
    const otherProps: StreamProps[] = []
    forEach(this.props.streams, (userStreams, userId) => {
      const localUser = userId === ME
      userStreams.streams.forEach((stream, i) => {
        const key = userStreams.userId + '_' + i
        const isActive = active === key
        const props: StreamProps = {
          active: isActive,
          key,
          stream: stream,
          userId: userStreams.userId,
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
    })
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
