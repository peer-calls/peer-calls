import * as constants from '../constants'
import { MaximizeAction, MinimizeToggleAction, StreamAction } from '../actions/StreamActions'

export type ActiveState = null | string

export type WindowState = undefined | 'minimized'

export interface WindowStates {
  // For example: `${peerId}_${index}`
  [streamKey: string]: WindowState
}

export function getStreamKey(peerId: string, streamId?: string) {
  return peerId + '_' + streamId
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
  const key = getStreamKey(action.payload.peerId, action.payload.streamId)
  return state[key] ? unminimize(state, key) : minimize(state, key)
}


function maximize(
  state: WindowStates,
  action: MaximizeAction,
): WindowStates {
  return action.payload.others.reduce((newState, stream) => {
    const key = getStreamKey(stream.peerId, stream.streamId)
    newState[key] = 'minimized'
    return newState
  }, {} as WindowStates)
}


export default function windowStates (
  state: WindowStates = {},
  action: StreamAction,
): WindowStates {
  switch (action.type) {
    case constants.MINIMIZE_TOGGLE:
      return minimizeToggle(state, action)
    case constants.MAXIMIZE:
      return maximize(state, action)
    default:
      return state
  }
}
