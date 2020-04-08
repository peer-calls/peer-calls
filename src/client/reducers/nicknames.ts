import { NICKNAMES_SET, PEER_REMOVE, ME, HANG_UP, NICKNAME_REMOVE } from '../constants'
import { NicknameActions, NicknameRemoveAction } from '../actions/NicknameActions'
import { RemovePeerAction } from '../actions/PeerActions'
import { nickname, userId } from '../window'
import omit from 'lodash/omit'
import { HangUpAction } from '../actions/CallActions'

export type Nicknames = Record<string, string>

const defaultState: Nicknames = {
  [ME]: getLocalNickname(),
}

export function getLocalNickname() {
  return localStorage && localStorage.nickname || nickname
}

function removeNickname(
  state: Nicknames,
  action: NicknameRemoveAction,
): Nicknames {
  const { userId } = action.payload
  const newState = {
    ...state,
  }
  if (userId !== ME) {
    delete newState[userId]
  }
  return newState
}

export default function nicknames(
  state = defaultState,
  action: NicknameActions | RemovePeerAction | HangUpAction,
) {
  switch (action.type) {
    case PEER_REMOVE:
      return omit(state, [action.payload.userId])
    case HANG_UP:
      return {[ME]: state[ME]}
    case NICKNAME_REMOVE:
      return removeNickname(state, action)
    case NICKNAMES_SET:
      return Object.keys(action.payload).reduce((obj, key) => {
        const value = action.payload[key]
        if (key === userId) {
          obj[ME] = value
        } else {
          obj[key] = value
        }
        return obj
      }, {[ME]: state[ME]} as Nicknames)
    default:
      return state
  }
}
