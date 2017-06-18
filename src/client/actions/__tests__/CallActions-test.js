jest.mock('../../socket.js')
jest.mock('../../window.js')
jest.mock('../../store.js')
jest.mock('../SocketActions.js')

import * as CallActions from '../CallActions.js'
import * as SocketActions from '../SocketActions.js'
import * as constants from '../../constants.js'
import socket from '../../socket.js'
import store from '../../store.js'
import { callId, getUserMedia } from '../../window.js'

jest.useFakeTimers()

describe('reducers/alerts', () => {

  beforeEach(() => {
    store.clearActions()
    getUserMedia.fail(false)
    SocketActions.handshake.mockReturnValue(jest.fn())
  })

  afterEach(() => {
    jest.runAllTimers()
    socket.removeAllListeners()
  })

  describe('init', () => {

    it('calls handshake.init when connected & got camera stream', done => {
      const promise = store.dispatch(CallActions.init())
      socket.emit('connect')
      expect(store.getActions()).toEqual([{
        type: constants.INIT_PENDING
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Connected to server socket',
          type: 'warning'
        }
      }])
      promise.then(() => {
        expect(SocketActions.handshake.mock.calls).toEqual([[{
          socket,
          roomName: callId,
          stream: getUserMedia.stream
        }]])
      })
      .then(done)
      .catch(done.fail)
    })

    it('calls dispatches disconnect message on disconnect', done => {
      const promise = store.dispatch(CallActions.init())
      socket.emit('connect')
      socket.emit('disconnect')
      expect(store.getActions()).toEqual([{
        type: constants.INIT_PENDING
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Connected to server socket',
          type: 'warning'
        }
      }, {
        type: constants.NOTIFY,
        payload: {
          id: jasmine.any(String),
          message: 'Server socket disconnected',
          type: 'error'
        }
      }])
      promise.then(done).catch(done.fail)
    })

    it('dispatches alert when failed to get media stream', done => {
      getUserMedia.fail(true)
      const promise = store.dispatch(CallActions.init())
      socket.emit('connect')
      promise
      .then(done.fail)
      .catch(err => {
        expect(err.message).toEqual('test')
        done()
      })
    })

  })

})
