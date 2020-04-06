import * as constants from '../constants'
import { MinimizeToggleAction, StreamAction } from '../actions/StreamActions'

export type ActiveState = null | string

export type WindowState = undefined | 'minimized'

export interface WindowStates {
  // For example: `${userId}_${index}`
  [streamKey: string]: WindowState
}

export function getStreamKey(userId: string, streamId?: string) {
  return userId + '_' + streamId
}

function unminimize(state: WindowStates, key: string): WindowStates {
  delete state[key]
  return {...state}
}

function minimize(state: WindowStates, key: string): WindowStates {
  return {
    ...state,
    [key]: 'minimized',
  }
}

function minimizeToggle(
  state: WindowStates,
  action: MinimizeToggleAction,
): WindowStates {
  const key = getStreamKey(action.payload.userId, action.payload.streamId)
  return state[key] ? unminimize(state, key) : minimize(state, key)
}


export default function windowStates (
  state: WindowStates = {},
  action: StreamAction,
): WindowStates {
  switch (action.type) {
    case constants.MINIMIZE_TOGGLE:
      return minimizeToggle(state, action)
    default:
      return state
  }
}
