jest.mock('../socket')
jest.mock('../window')
jest.mock('../store')
jest.mock('./SocketActions')

import * as CallActions from './CallActions'
import * as SocketActions from './SocketActions'
import * as constants from '../constants'
import socket from '../socket'
import storeMock from '../store'
import { callId, getUserMedia } from '../window'
import { MockStore } from 'redux-mock-store'

jest.useFakeTimers()

describe('reducers/alerts', () => {

  const store: MockStore = storeMock as any

  beforeEach(() => {
    store.clearActions();
    (getUserMedia as any).fail(false);
    (SocketActions.handshake as jest.Mock).mockReturnValue(jest.fn())
  })

  afterEach(() => {
    jest.runAllTimers()
    socket.removeAllListeners()
  })

  describe('init', () => {

    it('calls handshake.init when connected & got camera stream', async () => {
      const promise = CallActions.init(store.dispatch, store.getState)
      socket.emit('connect')
      expect(store.getActions()).toEqual([{
        type: constants.INIT_PENDING,
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Connected to server socket',
          type: 'warning',
        },
      }, {
        type: constants.MESSAGE_ADD,
        payload: {
          image: null,
          message: 'Connected to server socket',
          timestamp: jasmine.any(String),
          userId: '[PeerCalls]',
        },
      }])
      await promise
      expect((SocketActions.handshake as jest.Mock).mock.calls).toEqual([[{
        socket,
        roomName: callId,
        stream: (getUserMedia as any).stream,
      }]])
    })

    it('calls dispatches disconnect message on disconnect', async () => {

      const promise = CallActions.init(store.dispatch, store.getState)
      socket.emit('connect')
      socket.emit('disconnect')
      expect(store.getActions()).toEqual([{
        type: constants.INIT_PENDING,
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Connected to server socket',
          type: 'warning',
        },
      }, {
        type: constants.MESSAGE_ADD,
        payload: {
          image: null,
          message: 'Connected to server socket',
          timestamp: jasmine.any(String),
          userId: '[PeerCalls]',
        },
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Server socket disconnected',
          type: 'error',
        },
      }, {
        type: constants.MESSAGE_ADD,
        payload: {
          image: null,
          message: 'Server socket disconnected',
          timestamp: jasmine.any(String),
          userId: '[PeerCalls]',
        },
      }])
      await promise
    })

    it('dispatches alert when failed to get media stream', async () => {
      (getUserMedia as any).fail(true)
      const promise = CallActions.init(store.dispatch, store.getState)
      socket.emit('connect')
      await promise
    })

  })

})
