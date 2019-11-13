import * as constants from '../constants.js'
import _ from 'underscore'
import Peer from 'simple-peer'
import { PeerAction } from '../actions/PeerActions'

export interface PeersState {
  [userId: string]: Peer.Instance
}

const defaultState: PeersState = {}

export default function peers (state = defaultState, action: PeerAction) {
  switch (action.type) {
    case constants.PEER_ADD:
      return {
        ...state,
        [action.payload.userId]: action.payload.peer,
      }
    case constants.PEER_REMOVE:
      return _.omit(state, [action.payload.userId])
    case constants.PEERS_DESTROY:
      _.each(state, peer => peer.destroy())
      return defaultState
    default:
      return state
  }
}
