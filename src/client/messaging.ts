import forEach from 'lodash/forEach'
import { Middleware } from 'redux'
import { addMessage, MessageSendAction } from './actions/ChatActions'
import * as NotifyActions from './actions/NotifyActions'
import { Encoder } from './codec'
import { MESSAGE_SEND } from './constants'
import { State, Store } from './store'
import { TextEncoder } from './textcodec'
import { userId } from './window'

export function createMessagingMiddleware(): Middleware<Store, State> {
  return store => {
    const textEncoder = new TextEncoder()
    const encoder = new Encoder()

    encoder.on('data', event => {
      const { peers } = store.getState()
      forEach(peers, peer => {
        try {
          peer.send(event.chunk)
        } catch (err) {
          NotifyActions.error('Error sending message to peer: {0}', err)
        }
      })
    })

    encoder.on('error', event => {
      store.dispatch(
        NotifyActions.error('Error sending file: {0}', event.error))
    })

    return next => (action: MessageSendAction) => {
      switch (action.type) {
        case MESSAGE_SEND:
          encoder.encode({
            senderId: userId,
            data: textEncoder.encode(JSON.stringify(action.payload)),
          })
          return next(addMessage(action.payload))
        default:
          return next(action)
      }
    }
  }
}
