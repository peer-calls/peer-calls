jest.mock('../insertable-streams')
jest.mock('../window')

import * as StreamActions from '../actions/StreamActions'
import { createObjectURL, MediaStream, MediaStreamTrack, RTCRtpReceiver } from '../window'
import { MediaStreamAction } from '../actions/MediaActions'
import { MEDIA_STREAM } from '../constants'
import { createStore, Store } from '../store'
import { StreamsState } from './streams'

describe('reducers/alerts', () => {

  let store: Store, stream: MediaStream, peerId: string
  beforeEach(() => {
    store = createStore()
    peerId = 'testPeerId'
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
        pubStreamsKeysByPeerId: {},
        pubStreams: {},
        remoteStreamsKeysByClientId: {},
        remoteStreams: {},
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
          mirror: false,
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
          mirror: false,
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

  describe('addTrack', () => {
    it('creates a new stream and adds a track to it', () => {
      const track = new MediaStreamTrack()
      store.dispatch(StreamActions.addTrack({
        receiver: new RTCRtpReceiver(),
        streamId: 'stream-123',
        track,
        peerId,
      }))
      const { streams } = store.getState()
      const expected: StreamsState = {
        localStreams: {},
        pubStreamsKeysByPeerId: {},
        pubStreams: {},
        remoteStreamsKeysByClientId: {
          [peerId]: {
            'stream-123': undefined,
          },
        },
        remoteStreams: {
          'stream-123': {
            stream: jasmine.any(MediaStream) as any,
            streamId: 'stream-123',
            url: jasmine.any(String) as any,
          },
        },
      }
      expect(streams).toEqual(expected)
      const mediaStream = streams.remoteStreams['stream-123'].stream
      const tracks = mediaStream.getTracks()
      expect(tracks.length).toBe(1)
      expect(tracks[0]).toBe(track)
    })

    it('adds a track to existing stream', () => {
      const track1 = new MediaStreamTrack()
      const track2 = new MediaStreamTrack()
      store.dispatch(StreamActions.addTrack({
        receiver: new RTCRtpReceiver(),
        streamId: 'stream-123',
        track: track1,
        peerId,
      }))
      store.dispatch(StreamActions.addTrack({
        receiver: new RTCRtpReceiver(),
        streamId: 'stream-123',
        track: track2,
        peerId,
      }))
      const { streams } = store.getState()
      const expected: StreamsState = {
        localStreams: {},
        pubStreams: {},
        pubStreamsKeysByPeerId: {},
        remoteStreamsKeysByClientId: {
          [peerId]: {
            'stream-123': undefined,
          },
        },
        remoteStreams: {
          'stream-123': {
            stream: jasmine.any(MediaStream) as any,
            streamId: 'stream-123',
            url: jasmine.any(String) as any,
          },
        },
      }
      expect(streams).toEqual(expected)
      const mediaStream = streams.remoteStreams['stream-123'].stream
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
        receiver: new RTCRtpReceiver(),
        streamId: 'stream-1',
        track: track1,
        peerId,
      }))
    })

    it('removes a track from stream and removes stream', () => {
      store.dispatch(StreamActions.removeTrack({
        peerId,
        streamId: 'stream-1',
        track: track1,
      }))
      const { streams } = store.getState()
      const expected: typeof streams = {
        localStreams: {},
        pubStreamsKeysByPeerId: {},
        pubStreams: {},
        remoteStreamsKeysByClientId: {},
        remoteStreams: {},
      }
      expect(streams).toEqual(expected)
    })

    it('removes a track from stream when there are more tracks', () => {
      store.dispatch(StreamActions.addTrack({
        receiver: new RTCRtpReceiver(),
        streamId: 'stream-1',
        track: track2,
        peerId,
      }))
      store.dispatch(StreamActions.removeTrack({
        peerId,
        streamId: 'stream-1',
        track: track1,
      }))
      const { streams } = store.getState()
      const expected: typeof streams = {
        localStreams: {},
        pubStreamsKeysByPeerId: {},
        pubStreams: {},
        remoteStreamsKeysByClientId: {
          [peerId]: {
            'stream-1': undefined,
          },
        },
        remoteStreams: {
          'stream-1': {
            stream: jasmine.any(MediaStream) as any,
            streamId: 'stream-1',
            url: jasmine.any(String) as any,
          },
        },
      }
      expect(streams).toEqual(expected)
      const mediaStream = streams.remoteStreams['stream-1'].stream
      expect(mediaStream.getTracks()).toEqual([ track2 ])
    })
  })
})
