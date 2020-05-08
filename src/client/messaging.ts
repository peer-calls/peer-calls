import _debug from 'debug'
import forEach from 'lodash/forEach'
import { Middleware } from 'redux'
import { addMessage, MessageSendAction } from './actions/ChatActions'
import * as NotifyActions from './actions/NotifyActions'
import { Encoder } from './codec'
import { MESSAGE_SEND, ME } from './constants'
import { State, Store } from './store'
import { TextEncoder } from './textcodec'
import { userId } from './window'

const debug = _debug('peercalls')

export function createMessagingMiddleware(
  createEncoder = () => new Encoder(),
): Middleware<Store, State> {
  return store => {
    const textEncoder = new TextEncoder()
    const encoder = createEncoder()

    encoder.on('data', event => {
      const { peers } = store.getState()
      forEach(peers, (peer, id) => {
        try {
          debug('Send %d bytes to peer %s', event.chunk.byteLength, id)
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
          return next(addMessage({
            ...action.payload,
            userId: ME,
          }))
        default:
          return next(action)
      }
    }
  }
}
