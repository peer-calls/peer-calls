import { MESSAGE_ADD } from '../constants'

export interface MessageAddAction {
  type: 'MESSAGE_ADD'
  payload: Message
}

export interface Message {
  userId: string
  message: string
  timestamp: string
  image?: string
  // Indicates whether or not the message should be counted
  system?: boolean
}

export const addMessage = (message: Message): MessageAddAction => ({
  type: MESSAGE_ADD,
  payload: message,
})
