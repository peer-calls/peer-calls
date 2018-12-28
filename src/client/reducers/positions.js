import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'
import _ from 'underscore'

const defaultState = Immutable({})

export default function positions (state = defaultState, action) {
  switch (action && action.type) {
    case constants.POSITION_SET:
      return {
        ...state,
        [action.payload.userId]: action.payload.position
      }
    case constants.POSITION_REMOVE:
      return _.omit(state, [action.payload.userId])
    default:
      return state
  }
}
