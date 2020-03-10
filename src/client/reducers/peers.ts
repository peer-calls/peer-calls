import forEach from 'lodash/forEach'
import omit from 'lodash/omit'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'
import * as constants from '../constants'
import { MediaStreamAction } from '../actions/MediaActions'
import { RemoveStreamAction } from '../actions/StreamActions'

export type PeersState = Record<string, Peer.Instance>

const defaultState: PeersState = {}

let localStreams: Record<string, MediaStream> = {}

function handleRemoveStream(
  state: PeersState,
  action: RemoveStreamAction,
): PeersState {
  const stream = action.payload.stream
  if (action.payload.userId === constants.ME) {
    forEach(state, peer => {
      console.log('removing track from peer')
      stream.getTracks().forEach(track => {
        peer.removeTrack(track, stream)
      })
    })
  }

  return state
}

export default function peers(
  state = defaultState,
  action: PeerAction | MediaStreamAction | RemoveStreamAction,
): PeersState {
  switch (action.type) {
    case constants.PEER_ADD:
      return {
        ...state,
        [action.payload.userId]: action.payload.peer,
      }
    case constants.PEER_REMOVE:
      return omit(state, [action.payload.userId])
    case constants.PEERS_DESTROY:
      localStreams = {}
      forEach(state, peer => peer.destroy())
      return defaultState
    case constants.STREAM_REMOVE:
      return handleRemoveStream(state, action)
    case constants.MEDIA_STREAM:
      if (action.status === 'resolved') {
        forEach(state, peer => {
          const localStream = localStreams[action.payload.userId]
          localStream && localStream.getTracks().forEach(track => {
            peer.removeTrack(track, localStream)
          })
          const stream = action.payload.stream
          stream.getTracks().forEach(track => {
            peer.addTrack(track, stream)
          })
        })
        localStreams[action.payload.userId] = action.payload.stream
      }
      return state
    default:
      return state
  }
}
