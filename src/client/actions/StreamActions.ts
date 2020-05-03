import { MetadataPayload } from '../SocketEvent'
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
  userId: string
  streamId?: string
}

export interface MinimizeToggleAction {
  type: 'MINIMIZE_TOGGLE'
  payload: MinimizeTogglePayload
}

export interface RemoveTrackPayload {
  track: MediaStreamTrack
}

export interface RemoveTrackAction {
  type: 'PEER_STREAM_TRACK_REMOVE'
  payload: RemoveTrackPayload
}

export interface AddTrackPayload {
  mid: string
  streamId: string
  userId: string
  track: MediaStreamTrack
}

export interface AddTrackAction {
  type: 'PEER_STREAM_TRACK_ADD'
  payload: AddTrackPayload
}

export interface UserIdPayload {
  userId: string
}

export interface TracksMetadataAction {
  type: 'TRACKS_METADATA'
  payload: MetadataPayload
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

export const tracksMetadata = (
  payload: MetadataPayload,
): TracksMetadataAction => ({
  type: constants.TRACKS_METADATA,
  payload,
})

export type StreamAction =
  RemoveLocalStreamAction |
  MinimizeToggleAction |
  RemoveTrackAction |
  AddTrackAction
