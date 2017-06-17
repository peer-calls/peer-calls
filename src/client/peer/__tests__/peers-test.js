jest.mock('../../window/video.js')
jest.mock('../../callId.js')
jest.mock('../../iceServers.js')
jest.mock('../../store.js')
  // const configureStore = require('redux-mock-store').default
  // const { middlewares } = require('../../middlewares.js')
  // return configureStore(middlewares)({})
// })
jest.mock('simple-peer')
  // const EventEmitter = require('events').EventEmitter
  // const Peer = jest.genMockFunction().mockImplementation(() => {
  //   let peer = new EventEmitter()
  //   peer.destroy = jest.genMockFunction()
  //   peer.signal = jest.genMockFunction()
  //   Peer.instances.push(peer)
  //   return peer
  // })
  // Peer.instances = []
  // return Peer
// })

import * as constants from '../../constants.js'
import Peer from 'simple-peer'
import peers from '../peers.js'
import store from '../../store.js'
import { EventEmitter } from 'events'
import { play } from '../../window/video.js'

describe('peers', () => {
  function createSocket () {
    const socket = new EventEmitter()
    socket.id = 'user1'
    return socket
  }

  let socket, stream, user
  beforeEach(() => {
    store.clearActions()

    user = { id: 'user2' }
    socket = createSocket()
    Peer.instances = []
    Peer.mockClear()
    play.mockClear()
    stream = { stream: true }
  })

  const actions = {
    connecting: {
      type: constants.NOTIFY,
      payload: {
        message: 'Connecting to peer...',
        type: 'warning'
      }
    },
    established: {
      type: constants.NOTIFY,
      payload: {
        message: 'Peer connection established',
        type: 'warning'
      }
    }
  }

  afterEach(() => peers.clear())

  describe('create', () => {
    it('creates a new peer', () => {
      peers.create({ socket, user, initiator: 'user2', stream })

      expect(store.getActions()).toEqual([actions.connecting])

      expect(Peer.instances.length).toBe(1)
      expect(Peer.mock.calls.length).toBe(1)
      expect(Peer.mock.calls[0][0].initiator).toBe(false)
      expect(Peer.mock.calls[0][0].stream).toBe(stream)
    })

    it('sets initiator correctly', () => {
      peers.create({ socket, user, initiator: 'user1', stream })

      expect(Peer.instances.length).toBe(1)
      expect(Peer.mock.calls.length).toBe(1)
      expect(Peer.mock.calls[0][0].initiator).toBe(true)
      expect(Peer.mock.calls[0][0].stream).toBe(stream)
    })

    it('destroys old peer before creating new one', () => {
      peers.create({ socket, user, initiator: 'user2', stream })
      peers.create({ socket, user, initiator: 'user2', stream })

      expect(Peer.instances.length).toBe(2)
      expect(Peer.mock.calls.length).toBe(2)
      expect(Peer.instances[0].destroy.mock.calls.length).toBe(1)
      expect(Peer.instances[1].destroy.mock.calls.length).toBe(0)
    })
  })

  describe('events', () => {
    let peer

    beforeEach(() => {
      peers.create({ socket, user, initiator: 'user1', stream })
      peer = Peer.instances[0]
    })

    describe('connect', () => {
      beforeEach(() => peer.emit('connect'))

      it('sends a notification', () => {
        expect(store.getActions()).toEqual([
          actions.connecting,
          actions.established
        ])
      })

      it('dispatches "play" action', () => {
        expect(play.mock.calls.length).toBe(1)
      })
    })

    describe('data', () => {

      beforeEach(() => {
        window.TextDecoder = class TextDecoder {
          constructor(encoding) {
            this.encoding = encoding
          }
          decode(object) {
            return object.toString(this.encoding)
          }
        }
      })

      it('decodes a message', () => {
        store.clearActions()
        const message = 'test'
        const object = JSON.stringify({ message })
        peer.emit('data', Buffer.from(object, 'utf-8'))
        expect(store.getActions()).toEqual([{
          type: constants.NOTIFY,
          payload: {
            type: 'info',
            message: `${user.id}: ${message}`
          }
        }])
      })
    })
  })

  describe('get', () => {
    it('returns undefined when not found', () => {
      expect(peers.get(user.id)).not.toBeDefined()
    })

    it('returns Peer instance when found', () => {
      peers.create({ socket, user, initiator: 'user2', stream })

      expect(peers.get(user.id)).toBe(Peer.instances[0])
    })
  })

  describe('getIds', () => {
    it('returns ids of all peers', () => {
      peers.create({
        socket, user: { id: 'user2' }, initiator: 'user2', stream
      })
      peers.create({
        socket, user: { id: 'user3' }, initiator: 'user3', stream
      })

      expect(peers.getIds()).toEqual([ 'user2', 'user3' ])
    })
  })

  describe('destroy', () => {
    it('destroys a peer and removes it', () => {
      peers.create({ socket, user, initiator: 'user2', stream })

      peers.destroy(user.id)

      expect(Peer.instances[0].destroy.mock.calls.length).toEqual(1)
    })

    it('throws no error when peer missing', () => {
      peers.destroy('bla123')
    })
  })

  describe('clear', () => {
    it('destroys all peers and removes them', () => {
      peers.create({
        socket, user: { id: 'user2' }, initiator: 'user2', stream
      })
      peers.create({
        socket, user: { id: 'user3' }, initiator: 'user3', stream
      })

      peers.clear()

      expect(Peer.instances[0].destroy.mock.calls.length).toEqual(1)
      expect(Peer.instances[1].destroy.mock.calls.length).toEqual(1)

      expect(peers.getIds()).toEqual([])
    })
  })

  describe('message', () => {

    it('sends a message to all peers', () => {
      peers.create({
        socket, user: { id: 'user2' }, initiator: 'user2', stream
      })
      peers.create({
        socket, user: { id: 'user3' }, initiator: 'user3', stream
      })
      peers.message('test')
      expect(peers.get('user2').send.mock.calls)
      .toEqual([[ '{"message":"test"}' ]])
      expect(peers.get('user3').send.mock.calls)
      .toEqual([[ '{"message":"test"}' ]])
    })

  })
})
