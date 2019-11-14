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
    return undefined
  }
}

export interface StreamsState {
  [userId: string]: AddStreamPayload
}

function addStream (
  state: StreamsState, action: AddStreamAction,
): StreamsState {
  const { userId, stream } = action.payload

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
  state: StreamsState, action: RemoveStreamAction,
): StreamsState {
  const { userId } = action.payload
  const stream = state[userId]
  if (stream && stream.url) {
    revokeObjectURL(stream.url)
  }
  return omit(state, [userId])
}

export default function streams(
  state = defaultState,
    action: StreamAction,
): StreamsState {
  switch (action.type) {
    case STREAM_ADD:
      return addStream(state, action)
    case STREAM_REMOVE:
      return removeStream(state, action)
    default:
      return state
  }
}
