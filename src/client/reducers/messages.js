import * as constants from '../constants.js'
import Immutable from 'seamless-immutable'

const defaultState = Immutable([])

export default function messages (state = defaultState, action) {
  switch (action && action.type) {
    case constants.MESSAGE_ADD:
      const messages = state.asMutable()
      messages.push(action.payload)
      return Immutable(messages)
    default:
      return state
  }
}
