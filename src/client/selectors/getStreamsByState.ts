import forEach from 'lodash/forEach'
import map from 'lodash/map'
import { createSelector } from 'reselect'
import { StreamTypeCamera } from '../actions/StreamActions'
import { ME } from '../constants'
import { getNickname } from '../nickname'
import { LocalStream, StreamWithURL } from '../reducers/streams'
import { getStreamKey, WindowState } from '../reducers/windowStates'
import { State } from '../store'

export interface StreamProps {
  key: string
  stream?: StreamWithURL
  peerId: string
  muted?: boolean
  localUser?: boolean
  mirrored?: boolean
  windowState: WindowState
  nickname: string
}

function getWindowStates(state: State) {
  return state.windowStates
}

function getStreams(state: State) {
  return state.streams
}

function getNicknames(state: State) {
  return state.nicknames
}

export const getStreamsByState = createSelector(
  [ getWindowStates, getNicknames, getStreams ],
  (windowStates, nicknames, streams) => {
    const all: StreamProps[] = []
    const minimized: StreamProps[] = []
    const maximized: StreamProps[] = []

    function addStreamProps(props: StreamProps) {
      if (props.windowState === 'minimized') {
        minimized.push(props)
      } else {
        maximized.push(props)
      }

      all.push(props)
    }

    function isLocalStream(s: StreamWithURL): s is LocalStream {
      return 'mirror' in s && 'type' in s
    }

    function addStreamsByUser(
      localUser: boolean,
      peerId: string,
      streams: Array<StreamWithURL | LocalStream>,
    ) {

      if (!streams.length) {
        const key = getStreamKey(peerId, undefined)
        const props: StreamProps = {
          key,
          peerId,
          localUser,
          windowState: windowStates[key],
          nickname: getNickname(nicknames, peerId),
        }
        addStreamProps(props)
        return
      }

      streams.forEach((stream) => {
        const key = getStreamKey(peerId, stream.streamId)
        const props: StreamProps = {
          key,
          stream: stream,
          peerId,
          mirrored: localUser && isLocalStream(stream) &&
            stream.type === StreamTypeCamera && stream.mirror,
          muted: localUser,
          localUser,
          windowState: windowStates[key],
          nickname: getNickname(nicknames, peerId),
        }
        addStreamProps(props)
      })
    }

    const localStreams = map(streams.localStreams, s => s!)
    addStreamsByUser(true, ME, localStreams)

    forEach(nicknames, (_, peerId) => {
      if (peerId != ME) {
        const s = map(
          streams.pubStreamsKeysByPeerId[peerId],
          (_, streamId) => streams.pubStreams[streamId],
        )
        .map(pubStream => streams.remoteStreams[pubStream.streamId])
        .filter(s => !!s)

        addStreamsByUser(false, peerId, s)
      }
    })

    return { all, minimized, maximized }
  },
)
