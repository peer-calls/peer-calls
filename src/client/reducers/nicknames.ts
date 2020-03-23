import { NICKNAME_SET, PEER_REMOVE, ME } from '../constants'
import { NicknameActions } from '../actions/NicknameActions'
import { RemovePeerAction } from '../actions/PeerActions'
import { nickname } from '../window'
import omit from 'lodash/omit'

export type Nicknames = Record<string, string | undefined>

const defaultState: Nicknames = {
  [ME]: localStorage && localStorage.nickname || nickname,
}

export default function nicknames(
  state = defaultState,
    action: NicknameActions | RemovePeerAction,
) {
  switch (action.type) {
    case PEER_REMOVE:
      return omit(state, [action.payload.userId])
    case NICKNAME_SET:
      return {
        ...state,
        [action.payload.userId]: action.payload.nickname,
      }
    default:
      return state
  }
}
