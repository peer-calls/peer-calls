jest.mock('../../window.js')

import * as StreamActions from '../../actions/StreamActions.js'
import { applyMiddleware, createStore } from 'redux'
import { create } from '../../middlewares.js'
import reducers from '../index.js'

describe('reducers/alerts', () => {

  class MediaStream {}
  let store, stream, userId
  beforeEach(() => {
    store = createStore(
      reducers,
      applyMiddleware.apply(null, create())
    )
    userId = 'test id'
    stream = new MediaStream()
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
        [userId]: jasmine.any(String)
      })
    })
  })

  describe('removeStream', () => {
    it('removes a stream', () => {
      store.dispatch(StreamActions.addStream({ userId, stream }))
      store.dispatch(StreamActions.removeStream(userId))
      expect(store.getState().streams).toEqual({})
    })
  })

})
