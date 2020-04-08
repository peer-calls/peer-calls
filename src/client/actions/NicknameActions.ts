import { NICKNAMES_SET } from '../constants'

export interface NicknamesSetPayload {
  [userId: string]: string
}

export interface NicknamesSetAction {
  type: 'NICKNAMES_SET'
  payload: NicknamesSetPayload
}

export function setNicknames(payload: NicknamesSetPayload): NicknamesSetAction {
  return {
    type: NICKNAMES_SET,
    payload,
  }
}

export type NicknameActions = NicknamesSetAction
