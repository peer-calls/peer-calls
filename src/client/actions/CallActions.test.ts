jest.mock('../socket')
jest.mock('../window')
jest.mock('./SocketActions')

import * as CallActions from './CallActions'
import * as SocketActions from './SocketActions'
import * as constants from '../constants'
import socket from '../socket'
import { bindActionCreators, createStore, AnyAction, combineReducers, applyMiddleware } from 'redux'
import { middlewares } from '../middlewares'

import { Nicknames } from '../reducers/nicknames'

jest.useFakeTimers()

describe('CallActions', () => {

  let callActions: typeof CallActions

  function allActions(state: AnyAction[] = [], action: AnyAction) {
    return [...state, action]
  }

  let nicknamesState: Nicknames = {
    [constants.ME]: 'local-user',
  }

  let mediaState: {
    dialState: constants.DialState
  } = {
    dialState: constants.DIAL_STATE_HUNG_UP,
  }

  afterEach(() => {
    nicknamesState = {
      [constants.ME]: 'local-user',
    }

    mediaState = {
      dialState: constants.DIAL_STATE_HUNG_UP,
    }
  })

  function nicknames() {
    return nicknamesState
  }

  function media() {
    return mediaState
  }

  const configureStore = () => createStore(
    combineReducers({ media, nicknames, allActions }),
    applyMiddleware(...middlewares),
  )

  type Store = ReturnType<typeof configureStore>

  let store: Store

  const setup = () =>{
    store = configureStore()

    callActions = bindActionCreators(CallActions, store.dispatch);
    (SocketActions.handshake as jest.Mock).mockReturnValue(jest.fn())
  }

  afterEach(() => {
    jest.runAllTimers()
    socket.removeAllListeners()
  })

  describe('init', () => {

    it('dispatches init action when connected', async () => {
      setup()

      const promise = callActions.init()
      socket.emit('connect', undefined)
      await promise
      expect(store.getState().allActions.slice(1)).toEqual([{
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Connected to server socket',
          type: 'warning',
        },
      }, {
        type: constants.SOCKET_CONNECTED,
      }])
    })

    it('calls dispatches disconnect message on disconnect', async () => {
      setup()

      const promise = callActions.init()
      socket.emit('connect', undefined)
      socket.emit('disconnect', undefined)
      expect(store.getState().allActions.slice(1)).toEqual([{
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Connected to server socket',
          type: 'warning',
        },
      }, {
        type: constants.SOCKET_CONNECTED,
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Server socket disconnected',
          type: 'error',
        },
      }, {
        type: constants.SOCKET_DISCONNECTED,
      }])
      await promise
    })

    it('dispatches alert when failed to get media stream', async () => {
      setup()

      const promise = callActions.init()
      socket.emit('connect', undefined)
      await promise
    })

    describe('connect after in-call (server restart)', () => {
      beforeEach(() => {
        mediaState = {
          dialState: constants.DIAL_STATE_IN_CALL,
        }

        setup()
      })

      it('destroys all peers and initiates peer reconnection', async() => {
        const promise = callActions.init()

        socket.emit('connect', undefined)

        await promise

        expect(store.getState().allActions.slice(1)).toEqual([{
          payload: {
            id: jasmine.any(String),
            message: 'Connected to server socket',
            type: 'warning',
          },
          type: constants.NOTIFY,
        }, {
          type: constants.SOCKET_CONNECTED,
        }, {
          type: constants.PEER_REMOVE_ALL,
        }, {
          payload: {
            id: jasmine.any(String),
            message: 'Reconnecting to peer(s)...',
            type: 'info',
          },
          type: constants.NOTIFY,
        }, {
          status: 'pending',
          type: constants.DIAL,
        }])
      })
    })

  })

})
