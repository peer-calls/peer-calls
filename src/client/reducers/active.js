import * as constants from '../constants.js'

export default function active (state = null, action) {
  switch (action && action.type) {
    case constants.ACTIVE_SET:
      return action.payload.userId
    default:
      return state
  }
}
