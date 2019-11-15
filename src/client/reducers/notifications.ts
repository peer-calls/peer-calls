import * as constants from '../constants'
import { Notification, NotificationActionType } from '../actions/NotifyActions'

export type NotificationState = Record<string, Notification>

const defaultState: NotificationState = {}

export default function notifications (
  state = defaultState,
    action: NotificationActionType,
) {
  switch (action.type) {
    case constants.NOTIFY:
      return {
        ...state,
        [action.payload.id]: action.payload,
      }
    case constants.NOTIFY_DISMISS:
      return Object
      .keys(state)
      .filter(key => key !== action.payload.id)
      .reduce((obj, key) => {
        obj[key] = state[key]
        return obj
      }, {} as NotificationState)
    case constants.NOTIFY_CLEAR:
      return defaultState
    default:
      return state
  }
}
