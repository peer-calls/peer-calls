import * as constants from '../constants.js'
import _ from 'underscore'
import { createObjectURL, revokeObjectURL } from '../window.js'
import _debug from 'debug'

const debug = _debug('peercalls')
const defaultState = Object.freeze({})

function safeCreateObjectURL (stream) {
  try {
    return createObjectURL(stream)
  } catch (err) {
    debug('Error using createObjectURL: %s', err)
    return null
  }
}

function addStream (state, action) {
  const { userId, stream } = action.payload
  return Object.freeze({
    ...state,
    [userId]: Object.freeze({
      mediaStream: stream,
      url: safeCreateObjectURL(stream)
    })
  })
}

function removeStream (state, action) {
  const { userId } = action.payload
  const stream = state[userId]
  if (stream && stream.url) {
    revokeObjectURL(stream.url)
  }
  return Object.freeze(_.omit(state, [userId]))
}

export default function streams (state = defaultState, action) {
  switch (action && action.type) {
    case constants.STREAM_ADD:
      return addStream(state, action)
    case constants.STREAM_REMOVE:
      return removeStream(state, action)
    default:
      return state
  }
}
