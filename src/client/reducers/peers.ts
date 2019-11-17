import forEach from 'lodash/forEach'
import omit from 'lodash/omit'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'
import * as constants from '../constants'
import { MediaStreamAction } from '../actions/MediaActions'

export type PeersState = Record<string, Peer.Instance>

const defaultState: PeersState = {}

let localStream: MediaStream | undefined

export default function peers(
  state = defaultState,
  action: PeerAction | MediaStreamAction,
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
      localStream = undefined
      forEach(state, peer => peer.destroy())
      return defaultState
    case constants.MEDIA_STREAM:
      if (action.status === 'resolved') {
        forEach(state, peer => {
          localStream && peer.removeStream(localStream)
          peer.addStream(action.payload)
        })
        localStream = action.payload
      }
      return state
    default:
      return state
  }
}
