import * as constants from '../constants.js'

export default function active (state = null, action) {
  switch (action && action.type) {
    case constants.ACTIVE_SET:
    case constants.STREAM_ADD:
      return action.payload.userId
    case constants.ACTIVE_TOGGLE:
      return state === action.payload.userId ? null : action.payload.userId
    default:
      return state
  }
}
