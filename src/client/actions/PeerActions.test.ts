jest.mock('../window')
jest.mock('simple-peer')
jest.useFakeTimers()

import { EventEmitter } from 'events'
import { Dispatch } from 'redux'
import Peer from 'simple-peer'
import { Encoder } from '../codec'
import { HANG_UP, PEER_EVENT_DATA } from '../constants'
import { ClientSocket } from '../socket'
import { createStore, GetState, Store } from '../store'
import { TextEncoder } from '../textcodec'
import { MessageType } from './ChatActions'
import * as PeerActions from './PeerActions'

describe('PeerActions', () => {
  function createSocket () {
    const socket = new EventEmitter() as unknown as ClientSocket
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
      PeerActions.createPeer({ socket, user, initiator: false, stream })(
        dispatch, getState)

      expect(instances.length).toBe(1)
      expect(PeerMock.mock.calls.length).toBe(1)
      expect(PeerMock.mock.calls[0][0].initiator).toBe(false)
      expect(PeerMock.mock.calls[0][0].stream).toBe(stream)
    })

    it('sets initiator correctly', () => {
      PeerActions
      .createPeer({
        socket, user, initiator: true, stream,
      })(dispatch, getState)

      expect(instances.length).toBe(1)
      expect(PeerMock.mock.calls.length).toBe(1)
      expect(PeerMock.mock.calls[0][0].initiator).toBe(true)
      expect(PeerMock.mock.calls[0][0].stream).toBe(stream)
    })

    it('destroys old peer before creating new one', () => {
      PeerActions.createPeer({ socket, user, initiator: false, stream })(
        dispatch, getState)
      PeerActions.createPeer({ socket, user, initiator: true, stream })(
        dispatch, getState)

      expect(instances.length).toBe(2)
      expect(PeerMock.mock.calls.length).toBe(2)
      expect((instances[0].destroy as jest.Mock).mock.calls.length).toBe(1)
      expect((instances[1].destroy as jest.Mock).mock.calls.length).toBe(0)
    })
  })

  describe('events', () => {
    function createPeer() {
      PeerActions.createPeer({ socket, user, initiator: true, stream })(
        dispatch, getState)
      const peer = instances[instances.length - 1]
      return peer
    }

    describe('connect', () => {
      it('dispatches peer connection established message', () => {
        createPeer().emit('connect')
        // TODO
      })
    })

    describe('data', () => {

      it('decodes a message', async () => {
        const peer = createPeer()
        const message: MessageType = {
          timestamp: new Date().toISOString(),
          userId: 'test-user',
          type: 'text',
          payload: 'test',
        }
        const encoder = new Encoder()
        encoder.on('data', event => {
          peer.emit(PEER_EVENT_DATA, event.chunk)
        })
        const messageId = encoder.encode({
          senderId: user.id,
          data: new TextEncoder().encode(JSON.stringify(message)),
        })
        const promise = encoder.waitFor(messageId)
        jest.runAllTimers()
        await promise
        const { list } = store.getState().messages
        expect(list.length).toBeGreaterThan(0)
        expect(list[list.length - 1]).toEqual({
          userId: 'test-user',
          timestamp: new Date(message.timestamp).toLocaleString(),
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
      PeerActions.createPeer({ socket, user, initiator: false, stream })(
        dispatch, getState)

      const { peers } = store.getState()
      expect(peers[user.id]).toBe(instances[0])
    })
  })

  describe('destroyPeers', () => {
    it('destroys all peers and removes them', () => {
      PeerActions.createPeer({
        socket, user: { id: 'user2' }, initiator: true, stream,
      })(dispatch, getState)
      PeerActions.createPeer({
        socket, user: { id: 'user3' }, initiator: false, stream,
      })(dispatch, getState)

      store.dispatch({
        type: HANG_UP,
      })

      jest.runAllTimers()

      expect((instances[0].destroy as jest.Mock).mock.calls.length).toEqual(1)
      expect((instances[1].destroy as jest.Mock).mock.calls.length).toEqual(1)

      const { peers } = store.getState()
      expect(Object.keys(peers)).toEqual([])
    })
  })

})
