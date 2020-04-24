import * as constants from '../constants'
import { TrackMetadata } from '../../shared'

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

export interface RemoveTrackPayload {
  mid: string
  streamId: string
  userId: string
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
  payload: TrackMetadata[]
}

export const removeStream = (
  userId: string,
  stream: MediaStream,
): RemoveStreamAction => ({
  type: constants.STREAM_REMOVE,
  payload: { userId, stream },
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
  payload: TrackMetadata[],
): TracksMetadataAction => ({
  type: constants.TRACKS_METADATA,
  payload,
})

export type StreamAction =
  AddStreamAction |
  RemoveStreamAction |
  MinimizeToggleAction |
  RemoveTrackAction |
  AddTrackAction
