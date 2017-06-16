jest.mock('simple-peer')
jest.mock('../../store.js')
jest.mock('../../callId.js')
jest.mock('../../iceServers.js')

import * as constants from '../../constants.js'
import * as handshake from '../handshake.js'
import Peer from 'simple-peer'
import peers from '../peers.js'
import store from '../../store.js'
import { EventEmitter } from 'events'

describe('handshake', () => {
  let socket
  beforeEach(() => {
    socket = new EventEmitter()
    socket.id = 'a'

    Peer.instances = []
    store.clearActions()
  })

  afterEach(() => peers.clear())

  describe('socket events', () => {
    describe('users', () => {
      it('add a peer for each new user and destroy peers for missing', () => {
        handshake.init(socket, 'bla')

        // given
        let payload = {
          users: [{ id: 'a'}, { id: 'b' }],
          initiator: 'a'
        }
        socket.emit('users', payload)
        expect(Peer.instances.length).toBe(1)

        // when
        payload = {
          users: [{ id: 'a'}, { id: 'c' }],
          initiator: 'c'
        }
        socket.emit('users', payload)

        // then
        expect(Peer.instances.length).toBe(2)
        expect(Peer.instances[0].destroy.mock.calls.length).toBe(1)
        expect(Peer.instances[1].destroy.mock.calls.length).toBe(0)
      })
    })

    describe('signal', () => {
      let data
      beforeEach(() => {
        data = {}
        handshake.init(socket, 'bla')
        socket.emit('users', {
          initiator: 'a',
          users: [{ id: 'a' }, { id: 'b' }]
        })
      })

      it('should forward signal to peer', () => {
        socket.emit('signal', {
          userId: 'b',
          data
        })

        expect(Peer.instances.length).toBe(1)
        expect(Peer.instances[0].signal.mock.calls.length).toBe(1)
      })

      it('does nothing if no peer', () => {
        socket.emit('signal', {
          userId: 'a',
          data
        })

        expect(Peer.instances.length).toBe(1)
        expect(Peer.instances[0].signal.mock.calls.length).toBe(0)
      })
    })
  })

  describe('peer events', () => {
    let peer
    beforeEach(() => {
      let ready = false
      socket.once('ready', () => { ready = true })

      handshake.init(socket, 'bla')

      socket.emit('users', {
        initiator: 'a',
        users: [{ id: 'a' }, { id: 'b'}]
      })
      expect(Peer.instances.length).toBe(1)
      peer = Peer.instances[0]

      expect(ready).toBeDefined()
    })

    describe('error', () => {
      it('destroys peer', () => {
        peer.emit('error', new Error('bla'))
        expect(peer.destroy.mock.calls.length).toBe(1)
      })
    })

    describe('signal', () => {
      it('emits socket signal with user id', done => {
        let signal = { bla: 'bla' }

        socket.once('signal', payload => {
          expect(payload.userId).toEqual('b')
          expect(payload.signal).toBe(signal)
          done()
        })

        peer.emit('signal', signal)
      })
    })

    describe('stream', () => {
      it('adds a stream to streamStore', () => {
        store.clearActions()
        let stream = {}
        peer.emit('stream', stream)

        expect(store.getActions()).toEqual([{
          type: constants.STREAM_ADD,
          payload: {
            stream,
            userId: 'b'
          }
        }])
      })
    })

    describe('close', () => {
      it('removes stream from streamStore', () => {
        store.clearActions()
        peer.emit('close')

        expect(store.getActions()).toEqual([{
          type: constants.NOTIFY,
          payload: {
            message: 'Peer connection closed',
            type: 'error'
          }
        }, {
          type: constants.STREAM_REMOVE,
          payload: {
            userId: 'b'
          }
        }])
      })
    })
  })
})
