import * as constants from '../constants'
import { error, Notification, NotificationActionType } from '../actions/NotifyActions'
import { isRejectedAction } from '../async'
import { AnyAction } from 'redux'

export type NotificationState = Record<string, Notification>

const defaultState: NotificationState = {}

export default function notifications (
  state = defaultState,
  action: AnyAction,
) {
  if (isRejectedAction(action)) {
    action = error('' + action.payload)
  }
  return handleNotifications(state, action)
}

function handleNotifications (
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
