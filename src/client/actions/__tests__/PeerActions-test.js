jest.mock('../../window.js')
jest.mock('simple-peer')

import * as PeerActions from '../PeerActions.js'
import Peer from 'simple-peer'
import { EventEmitter } from 'events'
import { createStore } from '../../store.js'
import { play } from '../../window.js'

describe('PeerActions', () => {
  function createSocket () {
    const socket = new EventEmitter()
    socket.id = 'user1'
    return socket
  }

  let socket, stream, user, store
  beforeEach(() => {
    store = createStore()

    user = { id: 'user2' }
    socket = createSocket()
    Peer.instances = []
    Peer.mockClear()
    play.mockClear()
    stream = { stream: true }
  })

  describe('create', () => {
    it('creates a new peer', () => {
      store.dispatch(
        PeerActions.createPeer({ socket, user, initiator: 'user2', stream })
      )

      expect(Peer.instances.length).toBe(1)
      expect(Peer.mock.calls.length).toBe(1)
      expect(Peer.mock.calls[0][0].initiator).toBe(false)
      expect(Peer.mock.calls[0][0].stream).toBe(stream)
    })

    it('sets initiator correctly', () => {
      store.dispatch(
        PeerActions.createPeer({ socket, user, initiator: 'user1', stream })
      )

      expect(Peer.instances.length).toBe(1)
      expect(Peer.mock.calls.length).toBe(1)
      expect(Peer.mock.calls[0][0].initiator).toBe(true)
      expect(Peer.mock.calls[0][0].stream).toBe(stream)
    })

    it('destroys old peer before creating new one', () => {
      store.dispatch(
        PeerActions.createPeer({ socket, user, initiator: 'user2', stream })
      )
      store.dispatch(
        PeerActions.createPeer({ socket, user, initiator: 'user2', stream })
      )

      expect(Peer.instances.length).toBe(2)
      expect(Peer.mock.calls.length).toBe(2)
      expect(Peer.instances[0].destroy.mock.calls.length).toBe(1)
      expect(Peer.instances[1].destroy.mock.calls.length).toBe(0)
    })
  })

  describe('events', () => {
    let peer

    beforeEach(() => {
      store.dispatch(
        PeerActions.createPeer({ socket, user, initiator: 'user1', stream })
      )
      peer = Peer.instances[0]
    })

    describe('connect', () => {
      beforeEach(() => peer.emit('connect'))

      it('dispatches "play" action', () => {
        expect(play.mock.calls.length).toBe(1)
      })
    })

    describe('data', () => {

      beforeEach(() => {
        window.TextDecoder = class TextDecoder {
          constructor (encoding) {
            this.encoding = encoding
          }
          decode (object) {
            return object.toString(this.encoding)
          }
        }
      })

      it('decodes a message', () => {
        const message = 'test'
        const object = JSON.stringify({ message })
        peer.emit('data', Buffer.from(object, 'utf-8'))
        const { notifications } = store.getState()
        const keys = Object.keys(notifications)
        const n = notifications[keys[keys.length - 1]]
        expect(n).toEqual({
          id: jasmine.any(String),
          type: 'info',
          message: `${user.id}: ${message}`
        })
      })
    })
  })

  describe('get', () => {
    it('returns undefined when not found', () => {
      const { peers } = store.getState()
      expect(peers[user.id]).not.toBeDefined()
    })

    it('returns Peer instance when found', () => {
      store.dispatch(
        PeerActions.createPeer({ socket, user, initiator: 'user2', stream })
      )

      const { peers } = store.getState()
      expect(peers[user.id]).toBe(Peer.instances[0])
    })
  })

  describe('destroyPeers', () => {
    it('destroys all peers and removes them', () => {
      store.dispatch(PeerActions.createPeer({
        socket, user: { id: 'user2' }, initiator: 'user2', stream
      }))
      store.dispatch(PeerActions.createPeer({
        socket, user: { id: 'user3' }, initiator: 'user3', stream
      }))

      store.dispatch(PeerActions.destroyPeers())

      expect(Peer.instances[0].destroy.mock.calls.length).toEqual(1)
      expect(Peer.instances[1].destroy.mock.calls.length).toEqual(1)

      const { peers } = store.getState()
      expect(Object.keys(peers)).toEqual([])
    })
  })

  describe('sendMessage', () => {

    beforeEach(() => {
      store.dispatch(PeerActions.createPeer({
        socket, user: { id: 'user2' }, initiator: 'user2', stream
      }))
      store.dispatch(PeerActions.createPeer({
        socket, user: { id: 'user3' }, initiator: 'user3', stream
      }))
    })

    it('sends a message to all peers', () => {
      store.dispatch(PeerActions.sendMessage('test'))
      const { peers } = store.getState()
      expect(peers['user2'].send.mock.calls)
      .toEqual([[ '{"message":"test"}' ]])
      expect(peers['user3'].send.mock.calls)
      .toEqual([[ '{"message":"test"}' ]])
    })

  })
})
