import * as constants from '../constants.js'
import _ from 'underscore'

const defaultState = {}

export default function peers (state = defaultState, action) {
  switch (action && action.type) {
    case constants.PEER_ADD:
      return {
        ...state,
        [action.payload.userId]: action.payload.peer
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
