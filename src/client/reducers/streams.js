import * as constants from '../constants.js'
import createObjectURL from '../window/createObjectURL'
import Immutable from 'seamless-immutable'

const defaultState = Immutable({
  active: null,
  all: {}
})

function addStream (state, action) {
  const { userId, stream } = action.payload
  const all = state.all.merge({
    [userId]: {
      userId,
      url: createObjectURL(stream)
    }
  })
  return state.merge({ active: userId, all })
}

function removeStream (state, action) {
  const all = state.all.without(action.payload.userId)
  return state.merge({ all })
}

export default function streams (state = defaultState, action) {
  switch (action && action.type) {
    case constants.STREAM_ADD:
      return addStream(state, action)
    case constants.STREAM_ACTIVATE:
      return state.merge({ active: action.payload.userId })
    case constants.STREAM_REMOVE:
      return removeStream(state, action)
    default:
      return state
  }
}
