jest.mock('../window')

import * as StreamActions from '../actions/StreamActions'
import { createObjectURL, MediaStream, MediaStreamTrack } from '../window'
import { removeNickname } from '../actions/NicknameActions'
import { MediaStreamAction } from '../actions/MediaActions'
import { MEDIA_STREAM } from '../constants'
import { createStore, Store } from '../store'
import { StreamsState } from './streams'

describe('reducers/alerts', () => {

  let store: Store, stream: MediaStream, userId: string
  beforeEach(() => {
    store = createStore()
    userId = 'testUserId'
    stream = new MediaStream()
  })

  afterEach(() => {
    (createObjectURL as jest.Mock)
    .mockImplementation(object => 'blob://' + String(object))
  })

  describe('defaultState', () => {
    it('should have default state set', () => {
      expect(store.getState().streams).toEqual({
        localStreams: {},
        metadataByUserIdMid: {},
        streamsByUserId: {},
        trackIdToUserIdMid: {},
        tracksByUserIdMid: {},
      } as StreamsState)
    })
  })

  function createAddStreamAction(stream: MediaStream): MediaStreamAction {
    return {
      payload: {
        stream,
        type: StreamActions.StreamTypeCamera,
      },
      type: MEDIA_STREAM,
      status: 'resolved',
    }
  }
  describe('mediaStrea', () => {
    it('adds a stream', () => {
      store.dispatch(createAddStreamAction(stream))
      expect(store.getState().streams.localStreams).toEqual({
        [StreamActions.StreamTypeCamera]: {
          stream,
          streamId: stream.id,
          url: jasmine.any(String),
          type: StreamActions.StreamTypeCamera,
        },
      })
    })
    it('does not fail when createObjectURL fails', () => {
      (createObjectURL as jest.Mock)
      .mockImplementation(() => { throw new Error('test') })
      store.dispatch(createAddStreamAction(stream))
      expect(store.getState().streams.localStreams).toEqual({
        [StreamActions.StreamTypeCamera]: {
          stream,
          streamId: stream.id,
          type: StreamActions.StreamTypeCamera,
          url: undefined,
        },
      })
    })
  })

  describe('removeLocalStream', () => {
    it('removes a stream', () => {
      store.dispatch(createAddStreamAction(stream))
      store.dispatch(
        StreamActions
        .removeLocalStream(stream, StreamActions.StreamTypeCamera),
      )
      expect(store.getState().streams.localStreams).toEqual({})
    })
    it('does not fail when no stream', () => {
      store.dispatch(
        StreamActions
        .removeLocalStream(stream, StreamActions.StreamTypeCamera),
      )
    })
  })

  describe('removeNickname', () => {
    const otherUserId = 'other-user'
    const track1 = new MediaStreamTrack()
    const track2 = new MediaStreamTrack()
    const track3 = new MediaStreamTrack()
    beforeEach(() => {
      store.dispatch(StreamActions.addTrack({
        mid: '0',
        streamId: 'stream-1',
        track: track1,
        userId: otherUserId,
      }))
      store.dispatch(StreamActions.addTrack({
        mid: '1',
        streamId: 'stream-2',
        track: track2,
        userId: otherUserId,
      }))
      store.dispatch(StreamActions.addTrack({
        mid: '2',
        streamId: 'stream-3',
        track: track3,
        userId,
      }))
    })
    it('unassociates all tracks from user leaving', () => {
      store.dispatch(removeNickname({ userId: otherUserId }))
      const { streams } = store.getState()
      const users = Object.keys(streams.streamsByUserId)
      expect(users).toEqual([ userId ])
      const tracksByUserIdMid = streams.tracksByUserIdMid
      const expected: typeof tracksByUserIdMid = {
        [otherUserId + '::0']: {
          track: track1,
          mid: '0',
          association: undefined,
        },
        [otherUserId + '::1']: {
          track: track2,
          mid: '1',
          association: undefined,
        },
        [userId + '::2']: {
          track: track3,
          mid: '2',
          association: {
            streamId: 'stream-3',
            userId,
          },
        },
      }
      expect(tracksByUserIdMid).toEqual(expected)
    })
  })

  describe('addTrack', () => {
    it('creates a new stream and adds a track to it', () => {
      const track = new MediaStreamTrack()
      store.dispatch(StreamActions.addTrack({
        mid: '0',
        streamId: 'stream-123',
        track,
        userId,
      }))
      const { streams } = store.getState()
      const expected: StreamsState = {
        localStreams: {},
        metadataByUserIdMid: {},
        streamsByUserId: {
          [userId]: {
            userId,
            streams: [{
              stream: jasmine.any(MediaStream) as any,
              streamId: 'stream-123',
              url: jasmine.any(String) as any,
            }],
          },
        },
        trackIdToUserIdMid: {
          [track.id]: userId + '::0',
        },
        tracksByUserIdMid: {
          [userId + '::0']: {
            track,
            mid: '0',
            association: {
              userId,
              streamId: 'stream-123',
            },
          },
        },
      }
      expect(streams).toEqual(expected)
      const mediaStream = streams.streamsByUserId[userId].streams[0].stream
      const tracks = mediaStream.getTracks()
      expect(tracks.length).toBe(1)
      expect(tracks[0]).toBe(track)
    })

    it('adds a track to existing stream', () => {
      const track1 = new MediaStreamTrack()
      const track2 = new MediaStreamTrack()
      store.dispatch(StreamActions.addTrack({
        mid: '0',
        streamId: 'stream-123',
        track: track1,
        userId,
      }))
      store.dispatch(StreamActions.addTrack({
        mid: '1',
        streamId: 'stream-123',
        track: track2,
        userId,
      }))
      const { streams } = store.getState()
      const expected: StreamsState = {
        localStreams: {},
        metadataByUserIdMid: {},
        streamsByUserId: {
          [userId]: {
            userId,
            streams: [{
              stream: jasmine.any(MediaStream) as any,
              streamId: 'stream-123',
              url: jasmine.any(String) as any,
            }],
          },
        },
        trackIdToUserIdMid: {
          [track1.id]: userId + '::0',
          [track2.id]: userId + '::1',
        },
        tracksByUserIdMid: {
          [userId + '::0']: {
            track: track1,
            mid: '0',
            association: {
              userId,
              streamId: 'stream-123',
            },
          },
          [userId + '::1']: {
            track: track2,
            mid: '1',
            association: {
              userId,
              streamId: 'stream-123',
            },
          },
        },
      }
      expect(streams).toEqual(expected)
      const mediaStream = streams.streamsByUserId[userId].streams[0].stream
      const tracks = mediaStream.getTracks()
      expect(tracks.length).toBe(2)
      expect(tracks[0]).toBe(track1)
      expect(tracks[1]).toBe(track2)
    })
  })

  describe('removeTrack', () => {
    const track1 = new MediaStreamTrack()
    const track2 = new MediaStreamTrack()
    beforeEach(() => {
      stream = new MediaStream()
      store.dispatch(StreamActions.addTrack({
        mid: '0',
        streamId: 'stream-1',
        track: track1,
        userId,
      }))
    })

    it('removes a track from stream and removes stream', () => {
      store.dispatch(StreamActions.removeTrack({
        mid: '0',
        streamId: 'stream-1',
        track: track1,
        userId,
      }))
      const { streams } = store.getState()
      const expected: typeof streams = {
        localStreams: {},
        metadataByUserIdMid: {},
        streamsByUserId: {},
        trackIdToUserIdMid: {
          [track1.id]: userId + '::0',
        },
        tracksByUserIdMid: {
          [userId + '::0']: {
            track: track1,
            mid: '0',
            association: undefined,
          },
        },
      }
      expect(streams).toEqual(expected)
    })

    it('removes a track from stream when more tracks left', () => {
      store.dispatch(StreamActions.addTrack({
        mid: '1',
        streamId: 'stream-1',
        track: track2,
        userId,
      }))
      store.dispatch(StreamActions.removeTrack({
        mid: '0',
        streamId: 'stream-1',
        track: track1,
        userId,
      }))
      const { streams } = store.getState()
      const expected: typeof streams = {
        localStreams: {},
        metadataByUserIdMid: {},
        streamsByUserId: {
          [userId]: {
            streams: [{
              stream: jasmine.any(MediaStream) as any,
              streamId: 'stream-1',
              url: jasmine.any(String) as any,
            }],
            userId,
          },
        },
        trackIdToUserIdMid: {
          [track1.id]: userId + '::0',
          [track2.id]: userId + '::1',
        },
        tracksByUserIdMid: {
          [userId + '::0']: {
            track: track1,
            mid: '0',
            association: undefined,
          },
          [userId + '::1']: {
            track: track2,
            mid: '1',
            association: {
              streamId: 'stream-1',
              userId,
            },
          },
        },
      }
      expect(streams).toEqual(expected)
      const s = streams.streamsByUserId[userId].streams[0]
      expect(s.stream.getTracks()).toEqual([ track2 ])
    })
  })

})
