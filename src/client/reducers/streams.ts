import _ from 'underscore'
import { createObjectURL, revokeObjectURL } from '../window.js'
import _debug from 'debug'
import { AddStreamPayload, AddStreamAction, RemoveStreamAction, StreamAction } from '../actions/StreamActions.js'
import { STREAM_ADD, STREAM_REMOVE } from '../constants.js'

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
    [userId]: Object.freeze({
      mediaStream: stream,
      url: safeCreateObjectURL(stream),
    }),
  })
}

function removeStream (state: StreamsState, action: RemoveStreamAction) {
  const { userId } = action.payload
  const stream = state[userId]
  if (stream && stream.url) {
    revokeObjectURL(stream.url)
  }
  return Object.freeze(_.omit(state, [userId]))
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
