// import _debug from 'debug'
import forEach from 'lodash/forEach'
import keyBy from 'lodash/keyBy'
import omit from 'lodash/omit'
import { HangUpAction } from '../actions/CallActions'
import { MediaStreamAction } from '../actions/MediaActions'
import { AddStreamAction, AddTrackAction, RemoveStreamAction, RemoveTrackAction, StreamAction, StreamType, TracksMetadataAction } from '../actions/StreamActions'
import { HANG_UP, MEDIA_STREAM, STREAM_REMOVE, STREAM_TRACK_ADD, STREAM_TRACK_REMOVE, NICKNAME_REMOVE, TRACKS_METADATA } from '../constants'
import { createObjectURL, revokeObjectURL } from '../window'
import { NicknameRemoveAction, NicknameRemovePayload } from '../actions/NicknameActions'
import { TrackMetadata } from '../../shared'

// const debug = _debug('peercalls')
const defaultState = Object.freeze({
  localStreams: [],
  streamsByUserId: {},
  metadataByMid: {},
  trackIdToMid: {},
  tracksByMid: {},
})

function safeCreateObjectURL (stream: MediaStream) {
  try {
    return createObjectURL(stream)
  } catch (err) {
    return undefined
  }
}

export interface StreamWithURL {
  streamId: string
  stream: MediaStream
  type: StreamType | undefined
  url?: string
}

export interface UserStreams {
  userId: string
  streams: StreamWithURL[]
}

export interface StreamsState {
  localStreams: StreamWithURL[]
  streamsByUserId: Record<string, UserStreams>
  metadataByMid: Record<string, TrackMetadata>
  trackIdToMid: Record<string, string>
  tracksByMid: Record<string, TrackWithStreamId>
}

interface TrackWithStreamId {
  track: MediaStreamTrack
  association: TrackAssociation | undefined
}

interface TrackAssociation {
  streamId: string
  userId: string
}

interface MidWithUserId {
  mid: string
  streamId: string
  userId: string
}

interface StreamIdUserId {
  streamId: string
  userId: string
}


function getUserId(
  state: StreamsState,
  payload: MidWithUserId,
): StreamIdUserId {
  const { mid } = payload
  const metadata = state.metadataByMid[mid]

  if (metadata) {
    return {
      userId: metadata.userId,
      streamId: metadata.streamId,
    }
  }

  return {
    userId: payload.userId,
    streamId: payload.streamId,
  }
}

function addLocalStream (
  state: StreamsState, payload: AddStreamAction['payload'],
): StreamsState {
  const { stream } = payload

  const streamWithURL: StreamWithURL = {
    streamId: payload.stream.id,
    stream: payload.stream,
    type: payload.type,
    url: safeCreateObjectURL(stream),
  }

  return {
    ...state,
    localStreams: [...state.localStreams, streamWithURL],
  }
}

function removeLocalStream (
  state: StreamsState, payload: RemoveStreamAction['payload'],
): StreamsState {

  const localStreams = state.localStreams.filter(s => {
    if (s.stream === payload.stream) {
      s.stream.getTracks().forEach(track => track.stop())
      s.url && revokeObjectURL(s.url)
      return false
    }
    return true
  })

  return {
    ...state,
    localStreams,
  }
}

function removeTrack(
  state: StreamsState, payload: RemoveTrackAction['payload'],
): StreamsState {
  const { userId, streamId } = getUserId(state, payload)
  const { track } = payload

  const userStreams = state.streamsByUserId[userId]
  if (!userStreams) {
    return state
  }

  let streams = userStreams.streams
  const s = streams.find(s => s.streamId = streamId)

  if (!s) {
    return state
  }

  s.stream.removeTrack(track)
  if (s.stream.getTracks().length === 0) {
    s.url && revokeObjectURL(s.url)
    streams = streams.filter(_s => _s !== s)
  }

  if (streams.length > 0) {
    return {
      ...state,
      streamsByUserId: {
        ...state.streamsByUserId,
        [userId]: {
          ...userStreams,
          streams,
        },
      },
      tracksByMid: {
        ...state.tracksByMid,
        [payload.mid]: {
          track,
          association: undefined,
        },
      },
    }
  }

  return {
    ...state,
    streamsByUserId: omit(state.streamsByUserId, [userId]),
  }
}

function addTrack(
  state: StreamsState, payload: AddTrackAction['payload'],
): StreamsState {
  const { userId, streamId } = getUserId(state, payload)
  const { track } = payload

  const userStreams = state.streamsByUserId[userId] || {
    streams: [],
    userId,
  }

  const existing = userStreams.streams.find(s => s.streamId === streamId)
  if (existing) {
    existing.stream.addTrack(track)
    return state
  }

  const stream = new MediaStream()
  stream.addTrack(track)

  return {
    ...state,
    streamsByUserId: {
      ...state.streamsByUserId,
      [userId]: {
        ...userStreams,
        streams: [...userStreams.streams, {
          stream,
          streamId,
          url: safeCreateObjectURL(stream),
          type: undefined,
        }],
      },
    },
    trackIdToMid: {
      ...state.trackIdToMid,
      [track.id]: payload.mid,
    },
    tracksByMid: {
      ...state.tracksByMid,
      [payload.mid]: {
        track,
        association: {
          streamId,
          userId,
        },
      },
    },
  }
}

export function unassociateUserTracks(
  state: StreamsState,
  payload: NicknameRemovePayload,
): StreamsState  {
  const { userId } = payload

  const userStreams = state.streamsByUserId[userId]
  if (!userStreams) {
    return state
  }

  const tracksByMid: Record<string, TrackWithStreamId> = {}

  userStreams.streams.forEach(s => {
    s.stream.getTracks().forEach(track => {
      const mid = state.trackIdToMid[track.id]
      tracksByMid[mid] = {
        track,
        association: undefined,
      }
      s.stream.removeTrack(track)
    })
  })

  const streamsByUserId = omit(state.streamsByUserId, [userId])

  return {
    ...state,
    streamsByUserId,
    tracksByMid: {
      ...state.tracksByMid,
      ...tracksByMid,
    },
  }
}

function stopAllTracks(streams: StreamWithURL[]) {
  streams.forEach(s => {
    s.stream.getTracks().forEach(track => {
      track.stop()
      track.onmute = null
      track.onunmute = null
    })
  })
}

function setMetadata(
  state: StreamsState,
  metadata: TrackMetadata[],
): StreamsState {

  let newState = state

  metadata.forEach(m => {
    const { streamId, mid, userId } = m
    const t = state.tracksByMid[mid]

    if (!t) {
      // this track hasn't been seen yet so there's nothing to do
      return
    }

    if (!t.association) {
      // add the unassociated track
      newState = addTrack(newState, {
        mid,
        streamId,
        track: t.track,
        userId,
      })
      return
    }

    const a = t.association
    if (a.streamId === streamId && a.userId === userId) {
      // track is associated with the right userId / streamId
      return
    }

    newState = removeTrack(newState, {
      mid,
      streamId: a.streamId,
      track: t.track,
      userId: a.userId,
    })
    newState = addTrack(newState, {
      mid,
      streamId,
      track: t.track,
      userId,
    })
  })

  return {
    ...newState,
    metadataByMid: keyBy(metadata, 'mid'),
  }
}

export default function streams(
  state: StreamsState = defaultState,
  action:
    StreamAction |
    MediaStreamAction |
    HangUpAction |
    NicknameRemoveAction |
    TracksMetadataAction,
): StreamsState {
  switch (action.type) {
    case STREAM_REMOVE:
      return removeLocalStream(state, action.payload)
    case STREAM_TRACK_ADD:
      return addTrack(state, action.payload)
    case STREAM_TRACK_REMOVE:
      return removeTrack(state, action.payload)
    case NICKNAME_REMOVE:
      return unassociateUserTracks(state, action.payload)
    case TRACKS_METADATA:
      return setMetadata(state, action.payload)
    case HANG_UP:
      stopAllTracks(state.localStreams)
      forEach(state.streamsByUserId, us => stopAllTracks(us.streams))
      return defaultState
    case MEDIA_STREAM:
      if (action.status === 'resolved') {
        return addLocalStream(state, action.payload)
      } else {
        return state
      }
    default:
      return state
  }
}
