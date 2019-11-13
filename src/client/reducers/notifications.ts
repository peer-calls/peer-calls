import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'

const defaultState = Immutable({})

export default function notifications (state = defaultState, action) {
  switch (action && action.type) {
    case constants.NOTIFY:
      return state.merge({
        [action.payload.id]: action.payload
      })
    case constants.NOTIFY_DISMISS:
      return state.without(action.payload.id)
    case constants.NOTIFY_CLEAR:
      return defaultState
    default:
      return state
  }
}
