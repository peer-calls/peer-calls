import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'

const defaultState = Immutable([])

export default function notifications (state = defaultState, action) {
  switch (action && action.type) {
    case constants.NOTIFY:
      const notifications = state.asMutable()
      notifications.push(action.payload)
      return Immutable(notifications)
    case constants.NOTIFY_DISMISS:
      return state.filter(n => n !== action.payload)
    case constants.NOTIFY_CLEAR:
      return defaultState
    default:
      return state
  }
}
