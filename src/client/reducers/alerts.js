import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'

const defaultState = Immutable([])

export default function alerts (state = defaultState, action) {
  switch (action && action.type) {
    case constants.ALERT:
      const alerts = state.asMutable()
      alerts.push(action.payload)
      return Immutable(alerts)
    case constants.ALERT_DISMISS:
      return state.filter(a => a !== action.payload)
    case constants.ALERT_CLEAR:
      return defaultState
    default:
      return state
  }
}
