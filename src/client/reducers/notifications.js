import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'

const defaultState = Immutable({
  notifications: []
})

export default function notify (state = defaultState, action) {
  switch (action && action.type) {
    case constants.NOTIFY:
      const notifications = state.notifications.asMutable()
      notifications.push(action.payload)
      return state.merge({
        notifications
      })
    case constants.NOTIFY_DISMISS:
      return state.merge({
        notifications: state.notifications.filter(n => n !== action.payload)
      })
    case constants.NOTIFY_CLEAR:
      return state.merge({ notifications: [] })
    default:
      return state
  }
}
