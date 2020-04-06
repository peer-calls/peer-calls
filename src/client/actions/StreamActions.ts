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

export interface MinimizeTogglePayload {
  userId: string
  streamId?: string
}

export interface MinimizeToggleAction {
  type: 'MINIMIZE_TOGGLE'
  payload: MinimizeTogglePayload
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

export interface AddStreamTrackPayload {
  userId: string
  stream: MediaStream
  track: MediaStreamTrack
}

export interface AddStreamTrackAction {
  type: 'PEER_STREAM_TRACK_ADD'
  payload: AddStreamTrackPayload
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

export const addTrack = (
  payload: AddStreamTrackPayload,
): AddStreamTrackAction => ({
  type: constants.STREAM_TRACK_ADD,
  payload,
})

export const removeTrack = (
  payload: RemoveStreamTrackPayload,
): RemoveStreamTrackAction => ({
  type: constants.STREAM_TRACK_REMOVE,
  payload,
})

export const minimizeToggle = (
  payload: MinimizeTogglePayload,
): MinimizeToggleAction => ({
  type: constants.MINIMIZE_TOGGLE,
  payload,
})

export type StreamAction =
  AddStreamAction |
  RemoveStreamAction |
  MinimizeToggleAction |
  RemoveStreamTrackAction |
  AddStreamTrackAction
