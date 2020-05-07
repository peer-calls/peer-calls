import * as constants from '../constants'
import { MessageAddAction, MessageSendAction } from '../actions/ChatActions'
import { NotificationAddAction } from '../actions/NotifyActions'

export interface Message {
  userId: string
  message: string
  timestamp: string
  data?: string
  image?: boolean
  // Indicates whether or not the message should be counted
  system?: boolean
}

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
    system: true,
    timestamp: new Date().toLocaleString(),
  }
}

const imageRegexp = /^data:image\/(png|jpg|jpeg|gif);base64/

function handleMessage(
  state: MessagesState,
  action: MessageAddAction,
): MessagesState {
  const { payload } = action

  const count = state.count + 1

  switch (payload.type) {
    case 'file':
      return {
        ...state,
        count,
        list: [...state.list, {
          data: payload.payload.data,
          image: imageRegexp.test(payload.payload.data),
          userId: payload.userId,
          message: payload.payload.name,
          timestamp: new Date(payload.timestamp).toLocaleString(),
        }],
      }
    case 'text':
      return {
        ...state,
        count,
        list: [...state.list, {
          userId: payload.userId,
          message: payload.payload,
          timestamp: new Date(payload.timestamp).toLocaleString(),
        }],
      }
    default:
      return state
  }
}

export default function messages (
  state = defaultState,
    action: MessageAddAction | MessageSendAction | NotificationAddAction,
): MessagesState {
  switch (action.type) {
    case constants.NOTIFY:
      return {
        ...state,
        list: [...state.list, convertNotificationToMessage(action)],
      }
    case constants.MESSAGE_ADD:
      return handleMessage(state, action)
    default:
      return state
  }
}
