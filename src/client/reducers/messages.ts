import * as constants from '../constants'
import { Message, MessageAddAction } from '../actions/ChatActions'

export type MessagesState = Message[]

const defaultState: MessagesState = []

export default function messages (
  state = defaultState, action: MessageAddAction,
) {
  switch (action && action.type) {
    case constants.MESSAGE_ADD:
      return [...state, action.payload]
    default:
      return state
  }
}
