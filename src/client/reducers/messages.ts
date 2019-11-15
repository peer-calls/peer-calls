import * as constants from '../constants'
import { Message, MessageAddAction } from '../actions/ChatActions'
import { NotificationAddAction } from '../actions/NotifyActions'

export interface MessagesState {
  list: Message[]
  count: number
}

const defaultState: MessagesState = {
  list: [],
  count: 0,
}

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
      return {
        ...state,
        list: [...state.list, convertNotificationToMessage(action)],
      }
    case constants.MESSAGE_ADD:
      return {
        count: state.count + 1,
        list: [...state.list, action.payload],
      }
    default:
      return state
  }
}
