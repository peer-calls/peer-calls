import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'
import { createObjectURL } from '../window.js'

const defaultState = Immutable({})

function addStream (state, action) {
  const { userId, stream } = action.payload
  return state.merge({
    [userId]: createObjectURL(stream)
  })
}

const removeStream = (state, action) => state.without(action.payload.userId)

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
