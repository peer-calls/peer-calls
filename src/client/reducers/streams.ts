import _debug from 'debug'
import omit from 'lodash/omit'
import { AddStreamAction, RemoveStreamAction, StreamAction, StreamType, RemoveStreamTrackAction } from '../actions/StreamActions'
import { STREAM_ADD, STREAM_REMOVE, MEDIA_STREAM, STREAM_TRACK_REMOVE } from '../constants'
import { createObjectURL, revokeObjectURL } from '../window'
import { MediaStreamAction } from '../actions/MediaActions'

const debug = _debug('peercalls')
const defaultState = Object.freeze({})

function safeCreateObjectURL (stream: MediaStream) {
  try {
    return createObjectURL(stream)
  } catch (err) {
    debug('Error using createObjectURL: %s', err)
    return undefined
  }
}

export interface StreamWithURL {
  stream: MediaStream
  type: StreamType | undefined
  url?: string
}

export interface UserStreams {
  userId: string
  streams: StreamWithURL[]
}

export interface StreamsState {
  [userId: string]: UserStreams
}

function addStream (
  state: StreamsState, payload: AddStreamAction['payload'],
): StreamsState {
  const { userId, stream } = payload

  const userStreams = state[userId] || {
    userId,
    streams: [],
  }

  if (userStreams.streams.map(s => s.stream).indexOf(stream) >= 0) {
    return state
  }

  const streamWithURL: StreamWithURL = {
    stream,
    type: payload.type,
    url: safeCreateObjectURL(stream),
  }

  return {
    ...state,
    [userId]: {
      userId,
      streams: [...userStreams.streams, streamWithURL],
    },
  }
}

function removeStream (
  state: StreamsState, payload: RemoveStreamAction['payload'],
): StreamsState {
  const { userId, stream } = payload
  const userStreams = state[userId]
  if (!userStreams) {
    return state
  }

  if (stream) {
    const streams = userStreams.streams.filter(s => {
      const found = s.stream === stream
      if (found) {
        stream.getTracks().forEach(track => track.stop())
        s.url && revokeObjectURL(s.url)
      }
      return !found
    })
    if (streams.length > 0) {
      return {
        ...state,
        [userId]: {
          userId,
          streams,
        },
      }
    } else {
      omit(state, [userId])
    }
  }

  userStreams && userStreams.streams.forEach(s => {
    s.stream.getTracks().forEach(track => track.stop())
    s.url && revokeObjectURL(s.url)
  })
  return omit(state, [userId])
}

function removeStreamTrack(
  state: StreamsState, payload: RemoveStreamTrackAction['payload'],
): StreamsState {
  const { userId, stream, track } = payload
  const userStreams = state[userId]
  if (!userStreams) {
    return state
  }
  const index = userStreams.streams.map(s => s.stream).indexOf(stream)
  if (index < 0) {
    return state
  }
  stream.removeTrack(track)
  if (stream.getTracks().length === 0) {
    return removeStream(state, {userId, stream})
  }
  // UI does not update when a stream track is removed so there is no need to
  // update the state object
  return state
}

export default function streams(
  state = defaultState,
    action: StreamAction | MediaStreamAction,
): StreamsState {
  switch (action.type) {
    case STREAM_ADD:
      return addStream(state, action.payload)
    case STREAM_REMOVE:
      return removeStream(state, action.payload)
    case STREAM_TRACK_REMOVE:
      return removeStreamTrack(state, action.payload)
    case MEDIA_STREAM:
      if (action.status === 'resolved') {
        return addStream(state, action.payload)
      } else {
        return state
      }
    default:
      return state
  }
}
