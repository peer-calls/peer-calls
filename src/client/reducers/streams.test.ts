jest.mock('../window')

import * as StreamActions from '../actions/StreamActions'
import reducers from './index'
import { createObjectURL, MediaStream, MediaStreamTrack } from '../window'
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

  describe('addStreamTrack', () => {
    let stream: MediaStream
    beforeEach(() => {
      stream = new MediaStream()
      ;(stream.getTracks as jest.Mock).mockReturnValue([])
    })

    it('adds a stream when stream does not exist', () => {
      const track = new MediaStreamTrack()
      store.dispatch(StreamActions.addTrack({
        stream,
        track,
        userId,
      }))
      expect(store.getState().streams).toEqual({
        [userId]: {
          userId,
          streams: [{
            stream,
            url: jasmine.any(String),
          }],
        },
      })
    })

    it('adds a track to stream when track not added to stream', () => {
      const track = new MediaStreamTrack()
      store.dispatch(StreamActions.addTrack({
        stream,
        track,
        userId,
      }))
      expect((stream.addTrack as jest.Mock).mock.calls.length).toBe(1)
      expect((stream.addTrack as jest.Mock).mock.calls[0][0]).toBe(track)
    })

    it('adds stream and does not add existing track in stream', () => {
      const track = new MediaStreamTrack()
      ;(stream.getTracks as jest.Mock).mockReturnValue([track])
      store.dispatch(StreamActions.addTrack({
        stream,
        track,
        userId,
      }))
      expect((stream.addTrack as jest.Mock).mock.calls.length).toBe(0)
      expect(store.getState().streams).toEqual({
        [userId]: {
          userId,
          streams: [{
            stream,
            url: jasmine.any(String),
          }],
        },
      })
    })

    it('adds missing track to existing stream', () => {
      const track = new MediaStreamTrack()
      store.dispatch(StreamActions.addStream({
        stream,
        userId,
      }))
      store.dispatch(StreamActions.addTrack({
        stream,
        track,
        userId,
      }))
      expect((stream.addTrack as jest.Mock).mock.calls.length).toBe(1)
      expect((stream.addTrack as jest.Mock).mock.calls[0][0]).toBe(track)
      expect(store.getState().streams).toEqual({
        [userId]: {
          userId,
          streams: [{
            stream,
            url: jasmine.any(String),
          }],
        },
      })
    })
  })

  describe('removeStreamTrack', () => {
    let stream: MediaStream
    let tracks: MediaStreamTrack[]
    beforeEach(() => {
      stream = new MediaStream()
      store.dispatch(StreamActions.addStream({
        userId,
        stream,
      }))
      tracks = []
      ;(stream.getTracks as jest.Mock).mockImplementation(() => tracks)
      ;(stream.removeTrack as jest.Mock)
      .mockImplementation((track: MediaStreamTrack) => {
        tracks = tracks.filter(t => t !== track)
      })
    })

    it('removes a track from stream', () => {
      const track = new MediaStreamTrack()
      tracks = [track, new MediaStreamTrack()]
      store.dispatch(StreamActions.removeTrack({
        userId,
        stream,
        track,
      }))
      expect(store.getState().streams).toEqual({
        [userId]: {
          userId,
          streams: [{
            stream,
            url: jasmine.any(String),
          }],
        },
      })
    })

    it('removes a stream when no tracks left in stream', () => {
      const track = new MediaStreamTrack()
      tracks = [track]
      store.dispatch(StreamActions.removeTrack({
        userId,
        stream,
        track,
      }))
      expect(store.getState().streams).toEqual({})
    })
  })

})
