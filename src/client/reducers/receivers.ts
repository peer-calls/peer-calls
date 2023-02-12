import { omit } from 'lodash'
import { AddTrackAction, AddTrackPayload, RemoveTrackAction, RemoveTrackPayload } from '../actions/StreamActions'
import { STREAM_TRACK_ADD, STREAM_TRACK_REMOVE } from '../constants'

// receivers are indexed by streamId_trackId
export type ReceiversState = Record<string, RTCRtpReceiver>

const defaultState: ReceiversState = {}

export interface ReceiverStatsParams {
  // TODO include peerId/pubClientId.
  // The problem is the frontend keeps using peerId, and then
  // we use clientId as peerId in some places.
  streamId: string
  trackId: string
}

export function createReceiverStatsKey(
  params: ReceiverStatsParams,
): string {
  const { streamId, trackId } = params
  return `${streamId}_${trackId}`
}

function addTrack (
  state: Readonly<ReceiversState>,
  payload: AddTrackPayload,
): ReceiversState {
  const key = createReceiverStatsKey({
    streamId: payload.streamId,
    trackId: payload.track.id,
  })

  return {
    ...state,
   [key]: payload.receiver,
  }
}

function removeTrack (
  state: Readonly<ReceiversState>,
  payload: RemoveTrackPayload,
): ReceiversState {
  const key = createReceiverStatsKey({
    streamId: payload.streamId,
    trackId: payload.track.id,
  })

  return omit(state, key)
}

export default function streams(
  state: ReceiversState = defaultState,
  action:
    RemoveTrackAction |
    AddTrackAction,
): ReceiversState {
  switch (action.type) {
  case STREAM_TRACK_ADD:
    return addTrack(state, action.payload)
  case STREAM_TRACK_REMOVE:
    return removeTrack(state, action.payload)
  default:
    return state
  }
}
