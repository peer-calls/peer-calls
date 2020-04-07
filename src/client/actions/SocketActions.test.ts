jest.mock('simple-peer')
jest.mock('../window')

import * as SocketActions from './SocketActions'
import * as constants from '../constants'
import Peer from 'simple-peer'
import { EventEmitter } from 'events'
import { createStore, Store } from '../store'
import { ClientSocket } from '../socket'
import { MediaStream, MediaStreamTrack } from '../window'
import { SocketEvent } from '../../shared'

describe('SocketActions', () => {
  const roomName = 'bla'

  let socket: ClientSocket
  let store: Store
  let instances: Peer.Instance[]
  beforeEach(() => {
    socket = new EventEmitter() as any;
    (socket as any).id = 'a'

    store = createStore()

    instances = (Peer as any).instances = []
  })

  const userA = {
    socketId: 'socket-a',
    userId: 'user-a',
  }
  const userId = userA.userId

  const userB = {
    socketId: 'socket-b',
    userId: 'user-b',
  }

  const userC = {
    socketId: 'socket-c',
    userId: 'user-c',
  }

  describe('handshake', () => {
    describe('users', () => {
      beforeEach(() => {
        SocketActions.handshake({ socket, roomName, userId, store })
        const payload = {
          users: [userA, userB],
          initiator: userA.userId,
        }
        socket.emit('users', payload)
        expect(instances.length).toBe(1)
      })

      it('adds a peer for each new user and keeps active connections', () => {
        const payload = {
          users: [userA, userC],
          initiator:  userC.userId,
        }
        socket.emit(constants.SOCKET_EVENT_USERS, payload)

        // then
        expect(instances.length).toBe(2)
        expect((instances[0].destroy as jest.Mock).mock.calls.length).toBe(0)
        expect((instances[1].destroy as jest.Mock).mock.calls.length).toBe(0)
      })
    })

    describe('signal', () => {
      let data: Peer.SignalData
      beforeEach(() => {
        data = {} as any
        SocketActions.handshake({ socket, roomName, userId, store })
        socket.emit('users', {
          initiator: userA.userId,
          users: [userA, userB],
        })
      })

      it('should forward signal to peer', () => {
        socket.emit('signal', {
          userId: userB.userId,
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

      SocketActions.handshake({ socket, roomName, userId, store })

      socket.emit('users', {
        initiator: userA.userId,
        users: [userA, userB],
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

        socket.once('signal', (payload: SocketEvent['signal']) => {
          expect(payload.userId).toEqual(userB.userId)
          expect(payload.signal).toBe(signal)
          done()
        })

        peer.emit('signal', signal)
      })
    })

    describe('track unmute event', () => {
      it('adds a stream to streamStore', () => {
        const stream = new MediaStream()
        const track = new MediaStreamTrack()
        stream.addTrack(track)
        peer.emit(constants.PEER_EVENT_TRACK, track, stream)

        expect(track.onunmute).toBeDefined()
        // browsers should call onunmute after 'track' event, when track is
        // ready
        track.onunmute!(new Event('unmute'))

        expect(store.getState().streams).toEqual({
          [userB.userId]: {
            userId: userB.userId,
            streams: [{
              stream,
              type: undefined,
              url: jasmine.any(String),
            }],
          },
        })
      })
    })

    describe('track mute event', () => {
      it('removes track and stream from store', () => {
        const stream = new MediaStream()
        const track = new MediaStreamTrack()
        stream.addTrack(track)
        peer.emit(constants.PEER_EVENT_TRACK, track, stream)
        expect(track.onunmute).toBeDefined()
        track.onunmute!(new Event('unmute'))
        expect(track.onmute).toBeDefined()
        track.onmute!(new Event('mute'))
        expect(store.getState().streams).toEqual({})
      })
    })

    describe('close', () => {
      beforeEach(() => {
        const stream = new MediaStream()
        const track = new MediaStreamTrack()
        peer.emit(constants.PEER_EVENT_TRACK, track, stream)

        track.onunmute!(new Event('unmute'))
        track.onunmute!(new Event('unmute'))

        expect(stream.getTracks().length).toBe(1)
        expect(store.getState().streams).toEqual({
          [userB.userId]: {
            userId: userB.userId,
            streams: [{
              stream,
              type: undefined,
              url: jasmine.any(String),
            }],
          },
        })
      })

      it('removes stream & peer from store', () => {
        expect(store.getState().peers).toEqual({ [userB.userId]: peer })
        peer.emit('close')
        expect(store.getState().streams).toEqual({})
        expect(store.getState().peers).toEqual({})
      })
    })
  })
})
