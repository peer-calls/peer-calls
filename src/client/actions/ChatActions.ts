import * as NotifyActions from './NotifyActions'
import { Dispatch, GetState } from '../store'
import { MESSAGE_ADD, MESSAGE_SEND } from '../constants'
import { userId } from '../window'

export interface MessageAddAction {
  type: 'MESSAGE_ADD'
  payload: MessageType
}

export const addMessage = (message: MessageType): MessageAddAction => ({
  type: MESSAGE_ADD,
  payload: message,
})

export interface TextMessage {
  userId: string
  type: 'text'
  payload: string
  timestamp: string
}

export interface Base64File {
  name: string
  size: number
  type: string
  data: string
}

export interface FileMessage {
  userId: string
  type: 'file'
  payload: Base64File
  timestamp: string
}

export type MessageType = TextMessage | FileMessage

export interface MessageSendAction {
  type: 'MESSAGE_SEND'
  payload: MessageType
}


const sendMessage = (message: MessageType): MessageSendAction => {
  return {
    type: MESSAGE_SEND,
    payload: message,
  }
}

export const sendText = (payload: string) => {
  return sendMessage({
    payload,
    timestamp: new Date().toISOString(),
    type: 'text',
    userId,
  })
}

export const sendFile = (file: File) =>
async (dispatch: Dispatch, getState: GetState) => {
  const { name, size, type } = file
  if (!window.FileReader) {
    dispatch(NotifyActions.error('File API is not supported by your browser'))
    return
  }
  const reader = new window.FileReader()
  const base64File = await new Promise<Base64File>(resolve => {
    reader.addEventListener('load', () => {
      resolve({
        name,
        size,
        type,
        data: reader.result as string,
      })
    })
    reader.readAsDataURL(file)
  })

  dispatch(sendMessage({
    userId,
    payload: base64File,
    type: 'file',
    timestamp: new Date().toISOString(),
  }))
}
