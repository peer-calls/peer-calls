jest.mock('simple-peer')
jest.mock('../window')

import * as SocketActions from './SocketActions'
import * as constants from '../constants'
import Peer from 'simple-peer'
import { EventEmitter } from 'events'
import { createStore, Store, GetState } from '../store'
import { ClientSocket } from '../socket'
import { Dispatch } from 'redux'
import { MediaStream } from '../window'

describe('SocketActions', () => {
  const roomName = 'bla'

  let socket: ClientSocket
  let store: Store
  let dispatch: Dispatch
  let getState: GetState
  let instances: Peer.Instance[]
  beforeEach(() => {
    socket = new EventEmitter() as any;
    (socket as any).id = 'a'

    store = createStore()
    getState = store.getState
    dispatch = store.dispatch

    instances = (Peer as any).instances = []
  })

  describe('handshake', () => {
    describe('users', () => {
      beforeEach(() => {
        SocketActions.handshake({ socket, roomName })(dispatch, getState)
        const payload = {
          users: [{ id: 'a' }, { id: 'b' }],
          initiator: 'a',
        }
        socket.emit('users', payload)
        expect(instances.length).toBe(1)
      })

      it('adds a peer for each new user and destroys peers for missing', () => {
        const payload = {
          users: [{ id: 'a' }, { id: 'c' }],
          initiator: 'c',
        }
        socket.emit(constants.SOCKET_EVENT_USERS, payload)

        // then
        expect(instances.length).toBe(2)
        expect((instances[0].destroy as jest.Mock).mock.calls.length).toBe(1)
        expect((instances[1].destroy as jest.Mock).mock.calls.length).toBe(0)
      })
    })

    describe('signal', () => {
      let data: Peer.SignalData
      beforeEach(() => {
        data = {} as any
        SocketActions.handshake({ socket, roomName })(dispatch, getState)
        socket.emit('users', {
          initiator: 'a',
          users: [{ id: 'a' }, { id: 'b' }],
        })
      })

      it('should forward signal to peer', () => {
        socket.emit('signal', {
          userId: 'b',
          signal: data,
        })

        expect(instances.length).toBe(1)
        expect((instances[0].signal as jest.Mock).mock.calls.length).toBe(1)
      })

      it('does nothing if no peer', () => {
        socket.emit('signal', {
          userId: 'a',
          signal: data,
        })

        expect(instances.length).toBe(1)
        expect((instances[0].signal as jest.Mock).mock.calls.length).toBe(0)
      })
    })
  })

  describe('peer events', () => {
    let peer: Peer.Instance
    beforeEach(() => {
      let ready = false
      socket.once('ready', () => { ready = true })

      SocketActions.handshake({ socket, roomName })(dispatch, getState)

      socket.emit('users', {
        initiator: 'a',
        users: [{ id: 'a' }, { id: 'b' }],
      })
      expect(instances.length).toBe(1)
      peer = instances[0]

      expect(ready).toBeDefined()
    })

    describe('error', () => {
      it('destroys peer', () => {
        peer.emit(constants.PEER_EVENT_ERROR, new Error('bla'))
        expect((peer.destroy as jest.Mock).mock.calls.length).toBe(1)
      })
    })

    describe('signal', () => {
      it('emits socket signal with user id', done => {
        const signal = { bla: 'bla' }

        socket.once('signal', (payload: SocketActions.SignalOptions) => {
          expect(payload.userId).toEqual('b')
          expect(payload.signal).toBe(signal)
          done()
        })

        peer.emit('signal', signal)
      })
    })

    describe('stream', () => {
      it('adds a stream to streamStore', () => {
        const stream = {
          getTracks() {
            return [{
              stop: jest.fn(),
            }]
          },
        }
        peer.emit(constants.PEER_EVENT_STREAM, stream)

        expect(store.getState().streams).toEqual({
          b: {
            userId: 'b',
            stream,
            url: jasmine.any(String),
          },
        })
      })
    })

    describe('close', () => {
      beforeEach(() => {
        const stream = new MediaStream()
        peer.emit(constants.PEER_EVENT_STREAM, stream)
        expect(store.getState().streams).toEqual({
          b: {
            userId: 'b',
            stream,
            url: jasmine.any(String),
          },
        })
      })

      it('removes stream & peer from store', () => {
        expect(store.getState().peers).toEqual({ b: peer })
        peer.emit('close')
        expect(store.getState().streams).toEqual({})
        expect(store.getState().peers).toEqual({})
      })
    })
  })
})
