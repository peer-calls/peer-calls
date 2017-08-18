jest.mock('../../window.js')

import * as StreamActions from '../../actions/StreamActions.js'
import reducers from '../index.js'
import { createObjectURL, MediaStream } from '../../window.js'
import { applyMiddleware, createStore } from 'redux'
import { create } from '../../middlewares.js'

describe('reducers/alerts', () => {

  let store, stream, userId
  beforeEach(() => {
    store = createStore(
      reducers,
      applyMiddleware.apply(null, create())
    )
    userId = 'test id'
    stream = new MediaStream()
  })

  afterEach(() => {
    createObjectURL
    .mockImplementation(object => 'blob://' + String(object))
  })

  describe('defaultState', () => {
    it('should have default state set', () => {
      expect(store.getState().streams).toEqual({})
    })
  })

  describe('addStream', () => {
    it('adds a stream', () => {
      store.dispatch(StreamActions.addStream({ userId, stream }))
      expect(store.getState().streams).toEqual({
        [userId]: {
          mediaStream: stream,
          url: jasmine.any(String)
        }
      })
    })
    it('does not fail when createObjectURL fails', () => {
      createObjectURL
      .mockImplementation(() => { throw new Error('test') })
      store.dispatch(StreamActions.addStream({ userId, stream }))
      expect(store.getState().streams).toEqual({
        [userId]: {
          mediaStream: stream,
          url: null
        }
      })
    })
  })

  describe('removeStream', () => {
    it('removes a stream', () => {
      store.dispatch(StreamActions.addStream({ userId, stream }))
      store.dispatch(StreamActions.removeStream(userId))
      expect(store.getState().streams).toEqual({})
    })
    it('does not fail when no stream', () => {
      store.dispatch(StreamActions.removeStream(userId))
    })
  })

})
