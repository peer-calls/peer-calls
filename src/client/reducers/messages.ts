import * as constants from '../constants'
import { Message, MessageAddAction } from '../actions/ChatActions'
import { NotificationAddAction } from '../actions/NotifyActions'

export type MessagesState = Message[]

const defaultState: MessagesState = []

function convertNotificationToMessage(action: NotificationAddAction): Message {
  return {
    userId: '[PeerCalls]',
    message: action.payload.message,
    timestamp: new Date().toLocaleString(),
  }
}

export default function messages (
  state = defaultState, action: MessageAddAction | NotificationAddAction,
): MessagesState {
  switch (action.type) {
    case constants.NOTIFY:
      return [
      ...state,
      convertNotificationToMessage(action),
    ]
    case constants.MESSAGE_ADD:
      return [...state, action.payload]
    default:
      return state
  }
}
