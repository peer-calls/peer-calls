jest.mock('../window')

import * as StreamActions from '../actions/StreamActions'
import reducers from './index'
import { createObjectURL, MediaStream } from '../window'
import { applyMiddleware, createStore, Store } from 'redux'
import { create } from '../middlewares'

describe('reducers/alerts', () => {

  let store: Store, stream: MediaStream, userId: string
  beforeEach(() => {
    store = createStore(
      reducers,
      applyMiddleware(...create()),
    )
    userId = 'test id'
    stream = new MediaStream()
  })

  afterEach(() => {
    (createObjectURL as jest.Mock)
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
          userId,
          streams: [{
            stream,
            url: jasmine.any(String),
            type: undefined,
          }],
        },
      })
    })
    it('does not fail when createObjectURL fails', () => {
      (createObjectURL as jest.Mock)
      .mockImplementation(() => { throw new Error('test') })
      store.dispatch(StreamActions.addStream({ userId, stream }))
      expect(store.getState().streams).toEqual({
        [userId]: {
          userId,
          streams: [{
            stream,
            type: undefined,
            url: undefined,
          }],
        },
      })
    })
  })

  describe('removeStream', () => {
    it('removes a stream', () => {
      store.dispatch(StreamActions.addStream({ userId, stream }))
      store.dispatch(StreamActions.removeStream(userId, stream))
      expect(store.getState().streams).toEqual({})
    })
    it('does not fail when no stream', () => {
      store.dispatch(StreamActions.removeStream(userId, stream))
    })
  })

  describe('removeStreamTrack', () => {

  })

})
