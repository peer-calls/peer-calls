import forEach from 'lodash/forEach'
import omit from 'lodash/omit'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'
import * as constants from '../constants'

export type PeersState = Record<string, Peer.Instance>

const defaultState: PeersState = {}

export default function peers(
  state = defaultState,
  action: PeerAction,
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
      forEach(state, peer => peer.destroy())
      return defaultState
    default:
      return state
  }
}
