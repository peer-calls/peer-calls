import * as constants from '../constants'

export type StreamType = 'camera' | 'desktop'

export interface AddStreamPayload {
  userId: string
  type?: StreamType
  stream: MediaStream
}

export interface AddStreamAction {
  type: 'PEER_STREAM_ADD'
  payload: AddStreamPayload
}

export interface RemoveStreamAction {
  type: 'PEER_STREAM_REMOVE'
  payload: RemoveStreamPayload
}

export interface RemoveStreamPayload {
  userId: string
  stream: MediaStream
}

export interface SetActiveStreamPayload {
  userId: string
}

export interface SetActiveStreamAction {
  type: 'ACTIVE_SET'
  payload: SetActiveStreamPayload
}

export interface ToggleActiveStreamAction {
  type: 'ACTIVE_TOGGLE'
  payload: UserIdPayload
}

export interface RemoveStreamTrackPayload {
  userId: string
  stream: MediaStream
  track: MediaStreamTrack
}

export interface RemoveStreamTrackAction {
  type: 'PEER_STREAM_TRACK_REMOVE'
  payload: RemoveStreamTrackPayload
}

export interface UserIdPayload {
  userId: string
}

export const addStream = (payload: AddStreamPayload): AddStreamAction => ({
  type: constants.STREAM_ADD,
  payload,
})

export const removeStream = (
  userId: string,
  stream: MediaStream,
): RemoveStreamAction => ({
  type: constants.STREAM_REMOVE,
  payload: { userId, stream },
})

export const removeTrack = (
  payload: RemoveStreamTrackPayload,
): RemoveStreamTrackAction => ({
  type: constants.STREAM_TRACK_REMOVE,
  payload,
})

export const setActive = (userId: string): SetActiveStreamAction => ({
  type: constants.ACTIVE_SET,
  payload: { userId },
})

export const toggleActive = (userId: string): ToggleActiveStreamAction => ({
  type: constants.ACTIVE_TOGGLE,
  payload: { userId },
})

export type StreamAction =
  AddStreamAction |
  RemoveStreamAction |
  SetActiveStreamAction |
  ToggleActiveStreamAction |
  RemoveStreamTrackAction
