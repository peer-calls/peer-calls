jest.mock('simple-peer')

import { addPeer } from './actions/PeerActions'
import { createMessagingMiddleware } from './messaging'
import { createStore, applyMiddleware } from 'redux'
import reducers from './reducers'
import { sendText } from './actions/ChatActions'
import { Store } from './store'
import { deferred } from './deferred'
import { Encoder } from './codec'
import Peer from 'simple-peer'
import { ME } from './constants'

describe('messaging', () => {

  function configureStore(encoder: Encoder): Store {
    return createStore(
      reducers,
      applyMiddleware(createMessagingMiddleware(() => encoder)),
    )
  }

  describe('createMessagingMiddleware', () => {
    it('creates a middleware', () => {
      createStore(
        reducers,
        applyMiddleware(createMessagingMiddleware()),
      )
    })
  })

  describe('MESSAGE_SEND', () => {
    let store: Store
    let encoder: Encoder
    const userId1 = 'peer-a'
    const userId2 = 'peer-b'
    let peer1: Peer.Instance
    let peer2: Peer.Instance
    beforeEach(() => {
      encoder = new Encoder()
      store = configureStore(encoder)
      peer1 = new Peer()
      peer2 = new Peer()
      store.dispatch(addPeer({
        peer: peer1,
        userId: userId1,
      }))
      store.dispatch(addPeer({
        peer: peer2,
        userId: userId2,
      }))
    })

    function asMock(fn: (...args: any[]) => void): jest.Mock {
      return fn as jest.Mock
    }

    async function waitForMock(fn: (...args: any[]) => void) {
      const [ promise, resolve ] = deferred<void>()
      asMock(fn).mockImplementation(() => resolve())
      return promise
    }

    it('converts the action to add message', async () => {
      store.dispatch(sendText('hello'))
      expect(store.getState().messages).toEqual({
        count: 1,
        list: [{
          userId: ME,
          timestamp: jasmine.any(String),
          message: 'hello',
        }],
      })
    })

    it('sends chunks to all peers peer', async () => {
      const p1 = waitForMock(peer1.send)
      const p2 = waitForMock(peer2.send)
      const chunks: ArrayBuffer[] = []
      encoder.on('data', event => chunks.push(event.chunk))
      store.dispatch(sendText('hello'))
      await Promise.all([ p1, p2 ])
      expect(chunks.length).toBeGreaterThan(0)
      expect(asMock(peer1.send).mock.calls).toEqual(chunks.map(c => [c]))
      expect(asMock(peer2.send).mock.calls).toEqual(chunks.map(c => [c]))
    })

    describe('errors', () => {
      it('does not fail when sending to one peer errors out', async () => {
        asMock(peer1.send).mockImplementation(() => { throw Error('test') })
        const promise = waitForMock(peer2.send)
        const chunks: ArrayBuffer[] = []
        encoder.on('data', event => chunks.push(event.chunk))
        store.dispatch(sendText('hello'))
        await promise
        expect(chunks.length).toBeGreaterThan(0)
        expect(asMock(peer2.send).mock.calls).toEqual(chunks.map(c => [c]))
      })

      it('dispatches notification no encoder error', () => {
        encoder.emit('error', {
          messageId: 1,
          senderId: 'test',
          type: 'error',
          error: new Error('test'),
        })
        const { notifications } = store.getState()
        const n = Object.keys(notifications).map(k => notifications[k])
        expect(n).toEqual([{
          id: jasmine.any(String),
          message: 'Error sending file: Error: test',
          type: 'error',
        }])
      })
    })
  })

})
