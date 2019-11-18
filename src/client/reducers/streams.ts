import _debug from 'debug'
import omit from 'lodash/omit'
import { AddStreamAction, AddStreamPayload, RemoveStreamAction, StreamAction } from '../actions/StreamActions'
import { STREAM_ADD, STREAM_REMOVE, MEDIA_STREAM, ME } from '../constants'
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

export interface StreamsState {
  [userId: string]: AddStreamPayload
}

function addStream (
  state: StreamsState, payload: AddStreamAction['payload'],
): StreamsState {
  const { userId, stream } = payload

  const userStream: AddStreamPayload = {
    userId,
    stream,
    url: safeCreateObjectURL(stream),
  }

  return {
    ...state,
    [userId]: userStream,
  }
}

function removeStream (
  state: StreamsState, payload: RemoveStreamAction['payload'],
): StreamsState {
  const { userId } = payload
  const stream = state[userId]
  if (stream && stream.stream) {
    stream.stream.getTracks().forEach(track => track.stop())
  }
  if (stream && stream.url) {
    revokeObjectURL(stream.url)
  }
  return omit(state, [userId])
}

function replaceStream(state: StreamsState, stream: MediaStream): StreamsState {
  state = removeStream(state, {
    userId: ME,
  })
  return addStream(state, {
    userId: ME,
    stream,
  })
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
    case MEDIA_STREAM:
      if (action.status === 'resolved') {
        return replaceStream(state, action.payload)
      } else {
        return state
      }
    default:
      return state
  }
}
