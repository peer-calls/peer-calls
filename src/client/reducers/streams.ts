import _debug from 'debug'
import omit from 'lodash/omit'
import { AddStreamAction, AddStreamPayload, RemoveStreamAction, StreamAction } from '../actions/StreamActions'
import { STREAM_ADD, STREAM_REMOVE } from '../constants'
import { createObjectURL, revokeObjectURL } from '../window'

const debug = _debug('peercalls')
const defaultState = Object.freeze({})

function safeCreateObjectURL (stream: MediaStream) {
  try {
    return createObjectURL(stream)
  } catch (err) {
    debug('Error using createObjectURL: %s', err)
    return null
  }
}

export interface StreamsState {
  [userId: string]: AddStreamPayload
}

function addStream (state: StreamsState, action: AddStreamAction) {
  const { userId, stream } = action.payload
  return Object.freeze({
    ...state,
    [userId]: {
      mediaStream: stream,
      url: safeCreateObjectURL(stream),
    },
  })
}

function removeStream (state: StreamsState, action: RemoveStreamAction) {
  const { userId } = action.payload
  const stream = state[userId]
  if (stream && stream.url) {
    revokeObjectURL(stream.url)
  }
  return omit(state, [userId])
}

export default function streams (state = defaultState, action: StreamAction) {
  switch (action.type) {
    case STREAM_ADD:
      return addStream(state, action)
    case STREAM_REMOVE:
      return removeStream(state, action)
    default:
      return state
  }
}
