jest.mock('../socket')
jest.mock('../window')
jest.mock('./SocketActions')

import * as CallActions from './CallActions'
import * as SocketActions from './SocketActions'
import * as constants from '../constants'
import socket from '../socket'
import { bindActionCreators, createStore, AnyAction, combineReducers, applyMiddleware } from 'redux'
import reducers from '../reducers'
import { middlewares } from '../middlewares'

jest.useFakeTimers()

describe('CallActions', () => {

  let callActions: typeof CallActions

  function allActions(state: AnyAction[] = [], action: AnyAction) {
    return [...state, action]
  }

  const configureStore = () => createStore(
    combineReducers({...reducers, allActions }),
    applyMiddleware(...middlewares),
  )
  let store: ReturnType<typeof configureStore>

  beforeEach(() => {
    store = createStore(
      combineReducers({ allActions }),
      applyMiddleware(...middlewares),
    )
    callActions = bindActionCreators(CallActions, store.dispatch);
    (SocketActions.handshake as jest.Mock).mockReturnValue(jest.fn())
  })

  afterEach(() => {
    jest.runAllTimers()
    socket.removeAllListeners()
  })

  describe('init', () => {

    it('dispatches init action when connected', async () => {
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
      const promise = callActions.init()
      socket.emit('connect', undefined)
      await promise
    })

  })

})
