jest.mock('simple-peer')
jest.mock('../window')
// jest.mock('../actions/NicknameActions')

import * as NicknameActions from './NicknameActions'
import * as SocketActions from './SocketActions'
import * as constants from '../constants'
import Peer from 'simple-peer'
import { EventEmitter } from 'events'
import { createStore, Store } from '../store'
import { ClientSocket } from '../socket'
import { MediaStream, MediaStreamTrack } from '../window'
import { SocketEvent } from '../SocketEvent'
import { StreamsState } from '../reducers/streams'

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

  const peerA = 'peer-a'
  const userId = peerA

  const peerB = 'peer-b'
  const peerC = 'peer-c'

  const nickname = 'john'

  const nicknames: Record<string, string> = {
    [peerA]: 'user one',
    [peerB]: 'user two',
    [peerC]: 'user three',
  }

  describe('handshake', () => {
    describe('users', () => {
      beforeEach(() => {
        SocketActions.handshake({ nickname, socket, roomName, userId, store })
        const payload = {
          initiator: peerA,
          peerIds: [peerA, peerB],
          nicknames,
        }
        socket.emit('users', payload)
        expect(instances.length).toBe(1)
      })

      it('adds a peer for each new user and keeps active connections', () => {
        const payload = {
          peerIds: [peerA, peerC],
          initiator:  peerC,
          nicknames,
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
        SocketActions.handshake({ nickname, socket, roomName, userId, store })
        socket.emit('users', {
          initiator: peerA,
          peerIds: [peerA, peerB],
          nicknames,
        })
      })

      it('should forward signal to peer', () => {
        socket.emit('signal', {
          userId: peerB,
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

      SocketActions.handshake({ nickname, socket, roomName, userId, store })

      socket.emit('users', {
        initiator: peerA,
        peerIds: [peerA, peerB],
        nicknames,
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
          expect(payload.userId).toEqual(peerB)
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
        ;(track as any).muted = true
        stream.addTrack(track)
        peer.emit(constants.PEER_EVENT_TRACK, track, stream, '0')

        expect(track.onunmute).toBeDefined()
        // browsers should call onunmute after 'track' event, when track is
        // ready
        track.onunmute!(new Event('unmute'))
        const { streams } = store.getState()
        expect(streams).toEqual({
          localStreams: {},
          metadataByPeerIdMid: {},
          streamsByUserId: {
            [peerB]: {
              streams: [{
                stream: jasmine.any(MediaStream) as any,
                streamId: stream.id,
                url: jasmine.any(String) as any,
              }],
              userId: peerB,
            },
          },
          trackIdToPeerIdMid: {
            [track.id]: peerB + '::0',
          },
          tracksByPeerIdMid: {
            [peerB + '::0']: {
              association: {
                streamId: stream.id,
                userId: peerB,
              },
              mid: '0',
              track,
            },
          },
        } as StreamsState)
        const mediaStream = streams.streamsByUserId[peerB].streams[0].stream
        expect(mediaStream.getTracks()).toEqual([ track ])
      })
    })

    describe('track mute event', () => {
      it('removes track and stream from store', () => {
        const stream = new MediaStream()
        const track = new MediaStreamTrack()
        stream.addTrack(track)
        peer.emit(constants.PEER_EVENT_TRACK, track, stream, '0')
        expect(track.onunmute).toBeDefined()
        track.onunmute!(new Event('unmute'))
        expect(track.onmute).toBeDefined()
        track.onmute!(new Event('mute'))
        const { streams } = store.getState()
        expect(streams).toEqual({
          localStreams: {},
          metadataByPeerIdMid: {},
          streamsByUserId: {},
          trackIdToPeerIdMid: {
            [track.id]: peerB + '::0',
          },
          tracksByPeerIdMid: {
            [peerB + '::0']: {
              association: undefined,
              mid: '0',
              track,
            },
          },
        } as StreamsState)
      })
    })

    describe('hangUp', () => {
      beforeEach(() => {
        SocketActions.handshake({ nickname, socket, roomName, userId, store })
        store.dispatch(NicknameActions.setNicknames({
          a: 'A',
        }))
      })

      it('emits a removeNickname event', () => {
        socket.emit(constants.SOCKET_EVENT_HANG_UP, { userId: 'a' })
        expect(store.getState().nicknames).not.toHaveProperty('a')
      })
    })

    describe('close', () => {
      beforeEach(() => {
        const stream = new MediaStream()
        const track = new MediaStreamTrack()
        const mid = '0'
        peer.emit(constants.PEER_EVENT_TRACK, track, stream, mid)
        const streams = store.getState().streams.streamsByUserId
        expect(Object.keys(streams)).toEqual([ peerB ])
        const userStreams = streams[peerB].streams
        expect(userStreams.length).toBe(1)
        expect(userStreams[0].streamId).toBe(stream.id)
        expect(userStreams[0].stream).not.toBe(stream)
        expect(userStreams[0].stream.getTracks()).toEqual([ track ])
      })

      it('removes stream & peer from store', () => {
        expect(store.getState().peers).toEqual({ [peerB]: peer })
        peer.emit('close')
        expect(store.getState().streams).toEqual({
          localStreams: {},
          metadataByPeerIdMid: {},
          streamsByUserId: {},
          trackIdToPeerIdMid: {},
          tracksByPeerIdMid: {},
        } as StreamsState)
        expect(store.getState().peers).toEqual({})
      })
    })
  })
})
