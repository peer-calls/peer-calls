import * as constants from '../constants'
import { StreamAction } from '../actions/StreamActions'

export type ActiveState = null | string

export default function active (
  state: ActiveState = null,
  action: StreamAction,
): ActiveState {
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
