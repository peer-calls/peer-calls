import _debug from 'debug'
import forEach from 'lodash/forEach'
import omit from 'lodash/omit'
import { HangUpAction } from '../actions/CallActions'
import { MediaStreamAction } from '../actions/MediaActions'
import { AddStreamAction, AddStreamTrackAction, RemoveStreamAction, RemoveStreamTrackAction, StreamAction, StreamType } from '../actions/StreamActions'
import { HANG_UP, MEDIA_STREAM, STREAM_ADD, STREAM_REMOVE, STREAM_TRACK_ADD, STREAM_TRACK_REMOVE } from '../constants'
import { createObjectURL, revokeObjectURL } from '../window'

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

function addStreamTrack(
  state: StreamsState, payload: AddStreamTrackAction['payload'],
): StreamsState {
  const { userId, stream, track } = payload
  const userStreams = state[userId]
  const existingUserStream =
    userStreams && userStreams.streams.find(s => s.stream === stream)

  if (!stream.getTracks().includes(track)) {
    stream.addTrack(track)
  }

  if (!existingUserStream) {
    return addStream(state, {
      stream: payload.stream,
      userId: payload.userId,
    })
  }

  return state
}

export default function streams(
  state: StreamsState = defaultState,
  action: StreamAction | MediaStreamAction | HangUpAction,
): StreamsState {
  switch (action.type) {
    case STREAM_ADD:
      return addStream(state, action.payload)
    case STREAM_REMOVE:
      return removeStream(state, action.payload)
    case STREAM_TRACK_ADD:
      return addStreamTrack(state, action.payload)
    case STREAM_TRACK_REMOVE:
      return removeStreamTrack(state, action.payload)
    case HANG_UP:
      forEach(state, userStreams => {
        userStreams.streams.forEach(s => {
          s.stream.getTracks().forEach(track => {
            track.onmute = null
            track.onunmute = null
          })
        })
      })
      return defaultState
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
