jest.mock('../window')
jest.mock('simple-peer')

import * as PeerActions from './PeerActions'
import Peer from 'simple-peer'
import { EventEmitter } from 'events'
import { createStore, Store, GetState } from '../store'
import { Dispatch } from 'redux'
import { ClientSocket } from '../socket'

describe('PeerActions', () => {
  function createSocket () {
    const socket = new EventEmitter() as unknown as ClientSocket
    socket.id = 'user1'
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

    user = { id: 'user2' }
    socket = createSocket()
    instances = (Peer as any).instances = [];
    (Peer as unknown as jest.Mock).mockClear()
    stream = { stream: true } as unknown as MediaStream
    PeerMock = Peer as unknown as jest.Mock<Peer.Instance>
  })

  describe('create', () => {
    it('creates a new peer', () => {
      PeerActions.createPeer({ socket, user, initiator: 'user2', stream })(
        dispatch, getState)

      expect(instances.length).toBe(1)
      expect(PeerMock.mock.calls.length).toBe(1)
      expect(PeerMock.mock.calls[0][0].initiator).toBe(false)
      expect(PeerMock.mock.calls[0][0].stream).toBe(stream)
    })

    it('sets initiator correctly', () => {
      PeerActions
      .createPeer({
        socket, user, initiator: 'user1', stream,
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
    let peer: Peer.Instance

    beforeEach(() => {
      PeerActions.createPeer({ socket, user, initiator: 'user1', stream })(
        dispatch, getState)
      peer = instances[0]
    })

    describe('connect', () => {
      beforeEach(() => peer.emit('connect'))

      it('dispatches peer connection established message', () => {
        // TODO
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
        const payload = 'test'
        const object = JSON.stringify({ payload })
        peer.emit('data', Buffer.from(object, 'utf-8'))
        const { list } = store.getState().messages
        expect(list[list.length - 1]).toEqual({
          userId: 'user2',
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

    it('sends a message to all peers', () => {
      PeerActions.sendMessage({ payload: 'test', type: 'text' })(
        dispatch, getState)
      const { peers } = store.getState()
      expect((peers['user2'].send as jest.Mock).mock.calls)
      .toEqual([[ '{"payload":"test","type":"text"}' ]])
      expect((peers['user3'].send as jest.Mock).mock.calls)
      .toEqual([[ '{"payload":"test","type":"text"}' ]])
    })

  })
})
