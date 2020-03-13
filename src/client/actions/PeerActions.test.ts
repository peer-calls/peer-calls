jest.mock('../window')
jest.mock('simple-peer')

import * as PeerActions from './PeerActions'
import Peer from 'simple-peer'
import { EventEmitter } from 'events'
import { createStore, Store, GetState } from '../store'
import { Dispatch } from 'redux'
import { ClientSocket } from '../socket'
import { PEERCALLS, PEER_EVENT_DATA, ME } from '../constants'

describe('PeerActions', () => {
  function createSocket () {
    const socket = new EventEmitter() as unknown as ClientSocket
    socket.id = 'socket-id-user-1'
    return socket
  }

  let socket: ClientSocket
  let stream: MediaStream
  let user: { id: string }
  let store: Store
  let instances: Peer.Instance[]
  let dispatch: Dispatch
  let getState: GetState
  let PeerMock: jest.Mock<Peer.Instance>

  beforeEach(() => {
    store = createStore()
    dispatch = store.dispatch
    getState = store.getState

    user = { id: 'user1' }
    socket = createSocket()
    instances = (Peer as any).instances = [];
    (Peer as unknown as jest.Mock).mockClear()
    stream = { stream: true } as unknown as MediaStream
    PeerMock = Peer as unknown as jest.Mock<Peer.Instance>
  })

  describe('create', () => {
    it('creates a new peer', () => {
      PeerActions.createPeer({ socket, user, initiator: 'other-user', stream })(
        dispatch, getState)

      expect(instances.length).toBe(1)
      expect(PeerMock.mock.calls.length).toBe(1)
      expect(PeerMock.mock.calls[0][0].initiator).toBe(false)
      expect(PeerMock.mock.calls[0][0].stream).toBe(stream)
    })

    it('sets initiator correctly', () => {
      PeerActions
      .createPeer({
        socket, user, initiator: user.id, stream,
      })(dispatch, getState)

      expect(instances.length).toBe(1)
      expect(PeerMock.mock.calls.length).toBe(1)
      expect(PeerMock.mock.calls[0][0].initiator).toBe(true)
      expect(PeerMock.mock.calls[0][0].stream).toBe(stream)
    })

    it('destroys old peer before creating new one', () => {
      PeerActions.createPeer({ socket, user, initiator: 'user2', stream })(
        dispatch, getState)
      PeerActions.createPeer({ socket, user, initiator: 'user2', stream })(
        dispatch, getState)

      expect(instances.length).toBe(2)
      expect(PeerMock.mock.calls.length).toBe(2)
      expect((instances[0].destroy as jest.Mock).mock.calls.length).toBe(1)
      expect((instances[1].destroy as jest.Mock).mock.calls.length).toBe(0)
    })
  })

  describe('events', () => {
    function createPeer() {
      PeerActions.createPeer({ socket, user, initiator: 'user1', stream })(
        dispatch, getState)
      const peer = instances[instances.length - 1]
      return peer
    }

    describe('connect', () => {
      it('dispatches peer connection established message', () => {
        createPeer().emit('connect')
        // TODO
      })

      it('sends existing local streams to new peer', () => {
        PeerActions.sendMessage({
          payload: {nickname: 'john'},
          type: 'nickname',
        })(dispatch, getState)
        const peer = createPeer()
        peer.emit('connect')
      })

      it('sends current nickname to new peer', () => {
      })
    })

    describe('data', () => {

      beforeEach(() => {
        (window as any).TextDecoder = class TextDecoder {
          constructor (readonly encoding: string) {
          }
          decode (object: any) {
            return object.toString(this.encoding)
          }
        }
      })

      it('decodes a message', () => {
        const peer = createPeer()
        const message = {
          type: 'text',
          payload: 'test',
        }
        const object = JSON.stringify(message)
        peer.emit('data', Buffer.from(object, 'utf-8'))
        const { list } = store.getState().messages
        expect(list.length).toBeGreaterThan(0)
        expect(list[list.length - 1]).toEqual({
          userId: user.id,
          timestamp: jasmine.any(String),
          image: undefined,
          message: 'test',
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
      PeerActions.createPeer({ socket, user, initiator: 'user2', stream })(
        dispatch, getState)

      const { peers } = store.getState()
      expect(peers[user.id]).toBe(instances[0])
    })
  })

  describe('destroyPeers', () => {
    it('destroys all peers and removes them', () => {
      PeerActions.createPeer({
        socket, user: { id: 'user2' }, initiator: 'user2', stream,
      })(dispatch, getState)
      PeerActions.createPeer({
        socket, user: { id: 'user3' }, initiator: 'user3', stream,
      })(dispatch, getState)

      store.dispatch(PeerActions.destroyPeers())

      expect((instances[0].destroy as jest.Mock).mock.calls.length).toEqual(1)
      expect((instances[1].destroy as jest.Mock).mock.calls.length).toEqual(1)

      const { peers } = store.getState()
      expect(Object.keys(peers)).toEqual([])
    })
  })

  describe('sendMessage', () => {

    beforeEach(() => {
      PeerActions.createPeer({
        socket, user: { id: 'user2' }, initiator: 'user2', stream,
      })(dispatch, getState)
      PeerActions.createPeer({
        socket, user: { id: 'user3' }, initiator: 'user3', stream,
      })(dispatch, getState)
    })

    it('sends a text message to all peers', () => {
      PeerActions.sendMessage({ payload: 'test', type: 'text' })(
        dispatch, getState)
      const { peers } = store.getState()
      expect((peers['user2'].send as jest.Mock).mock.calls)
      .toEqual([[ '{"payload":"test","type":"text"}' ]])
      expect((peers['user3'].send as jest.Mock).mock.calls)
      .toEqual([[ '{"payload":"test","type":"text"}' ]])
    })

    it('sends a nickname change to all peers', () => {
      PeerActions.sendMessage({
        payload: {nickname: 'john'},
        type: 'nickname',
      })(dispatch, getState)
      const { nicknames, peers } = store.getState()
      expect((peers['user2'].send as jest.Mock).mock.calls)
      .toEqual([[ '{"payload":{"nickname":"john"},"type":"nickname"}' ]])
      expect((peers['user3'].send as jest.Mock).mock.calls)
      .toEqual([[ '{"payload":{"nickname":"john"},"type":"nickname"}' ]])
      expect(nicknames[ME]).toBe('john')
    })

  })

  describe('receive message (handleData)', () => {
    let peer: Peer.Instance
    function emitData(message: PeerActions.Message) {
      peer.emit(PEER_EVENT_DATA, JSON.stringify(message))
    }
    beforeEach(() => {
      PeerActions.createPeer({
        socket, user: { id: 'user2' }, initiator: 'user2', stream,
      })(dispatch, getState)
      peer = store.getState().peers['user2']
    })

    it('handles a message', () => {
      emitData({
        payload: 'hello',
        type: 'text',
      })
      expect(store.getState().messages.list)
      .toEqual([{
        message: 'Connecting to peer...',
        userId: PEERCALLS,
        timestamp: jasmine.any(String),
      }, {
        message: 'hello',
        userId: 'user2',
        image: undefined,
        timestamp: jasmine.any(String),
      }])
    })

    it('handles nickname changes', () => {
      emitData({
        payload: {nickname: 'john'},
        type: 'nickname',
      })
      emitData({
        payload: {nickname: 'john2'},
        type: 'nickname',
      })
      expect(store.getState().messages.list)
      .toEqual([{
        message: 'Connecting to peer...',
        userId: PEERCALLS,
        timestamp: jasmine.any(String),
      }, {
        message: 'User user2 is now known as john',
        userId: PEERCALLS,
        image: undefined,
        timestamp: jasmine.any(String),
      }, {
        message: 'User john is now known as john2',
        userId: PEERCALLS,
        image: undefined,
        timestamp: jasmine.any(String),
      }])
    })
  })
})
