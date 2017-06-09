import * as constants from '../constants.js'
import createObjectURL from '../browser/createObjectURL'
import Immutable from 'seamless-imutable'

const defaultState = Immutable({
  active: null,
  streams: {}
})

function addStream (state, action) {
  const { userId, stream } = action.payload
  const streams = state.streams.merge({
    [userId]: {
      userId,
      stream,
      url: createObjectURL(stream)
    }
  })
  return { active: userId, streams }
}

function removeStream (state, action) {
  const streams = state.streams.without(action.payload.userId)
  return state.merge({ streams })
}

export default function stream (state = defaultState, action) {
  switch (action && action.type) {
    case constants.STREAM_ADD:
      return addStream(state, action)
    case constants.STREAM_REMOVE:
      return removeStream(state, action)
    default:
      return state
  }
}
