import * as constants from '../constants'
import { AlertActionType, Alert } from '../actions/NotifyActions'

export type AlertState = Alert[]

const defaultState: AlertState = []

export default function alerts (state = defaultState, action: AlertActionType) {
  switch (action.type) {
    case constants.ALERT:
      return [...state, action.payload]
    case constants.ALERT_DISMISS:
      return state.filter(a => a !== action.payload)
    case constants.ALERT_CLEAR:
      return defaultState
    default:
      return state
  }
}
