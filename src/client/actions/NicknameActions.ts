import { NICKNAME_SET } from '../constants'

export interface NicknameSetPayload {
  nickname: string
  userId: string
}

export interface NicknameSetAction {
  type: 'NICKNAME_SET'
  payload: NicknameSetPayload
}

export function setNickname(payload: NicknameSetPayload): NicknameSetAction {
  return {
    type: NICKNAME_SET,
    payload,
  }
}

export type NicknameActions = NicknameSetAction
