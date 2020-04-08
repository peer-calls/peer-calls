import { NICKNAME_REMOVE, NICKNAMES_SET } from '../constants'

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

export interface NicknameRemovePayload {
  userId: string
}

export interface NicknameRemoveAction {
  type: 'NICKNAME_REMOVE'
  payload: NicknameRemovePayload
}

export function removeNickname(
  payload: NicknameRemovePayload,
): NicknameRemoveAction {
  return {
    type: NICKNAME_REMOVE,
    payload,
  }
}

export type NicknameActions = NicknamesSetAction | NicknameRemoveAction
