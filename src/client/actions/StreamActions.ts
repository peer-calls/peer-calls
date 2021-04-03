import { PubTrackEvent } from '../SocketEvent'
import * as constants from '../constants'

export type StreamType = 'camera' | 'desktop'
export const StreamTypeCamera: StreamType = 'camera'
export const StreamTypeDesktop: StreamType = 'desktop'

export interface AddLocalStreamPayload {
  type: StreamType
  stream: MediaStream
}

export interface RemoveLocalStreamAction {
  type: 'PEER_STREAM_REMOVE'
  payload: RemoveLocalStreamPayload
}

export interface RemoveLocalStreamPayload {
  stream: MediaStream
  streamType: StreamType
}

export interface MinimizeTogglePayload {
  peerId: string
  streamId?: string
}

export interface MinimizeToggleAction {
  type: 'MINIMIZE_TOGGLE'
  payload: MinimizeTogglePayload
}

export interface RemoveTrackPayload {
  streamId: string
  peerId: string
  track: MediaStreamTrack
}

export interface RemoveTrackAction {
  type: 'PEER_STREAM_TRACK_REMOVE'
  payload: RemoveTrackPayload
}

export interface AddTrackPayload {
  streamId: string
  peerId: string
  track: MediaStreamTrack
}

export interface AddTrackAction {
  type: 'PEER_STREAM_TRACK_ADD'
  payload: AddTrackPayload
}

export interface PeerIdPayload {
  peerId: string
}

export interface PubTrackEventAction {
  type: 'PUB_TRACK_EVENT'
  payload: PubTrackEvent
}

export const removeLocalStream = (
  stream: MediaStream,
  streamType: StreamType,
): RemoveLocalStreamAction => ({
  type: constants.STREAM_REMOVE,
  payload: { stream, streamType },
})

export const addTrack = (
  payload: AddTrackPayload,
): AddTrackAction => ({
  type: constants.STREAM_TRACK_ADD,
  payload,
})

export const removeTrack = (
  payload: RemoveTrackPayload,
): RemoveTrackAction => ({
  type: constants.STREAM_TRACK_REMOVE,
  payload,
})

export const minimizeToggle = (
  payload: MinimizeTogglePayload,
): MinimizeToggleAction => ({
  type: constants.MINIMIZE_TOGGLE,
  payload,
})

export const pubTrackEvent = (
  payload: PubTrackEvent,
): PubTrackEventAction => ({
  type: constants.PUB_TRACK_EVENT,
  payload,
})

export type StreamAction =
  RemoveLocalStreamAction |
  MinimizeToggleAction |
  RemoveTrackAction |
  AddTrackAction
