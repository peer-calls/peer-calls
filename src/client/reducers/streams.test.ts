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
        metadataByPeerIdMid: {},
        streamsByUserId: {},
        trackIdToPeerIdMid: {},
        tracksByPeerIdMid: {},
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
      const tracksByPeerIdMid = streams.tracksByPeerIdMid
      const expected: typeof tracksByPeerIdMid = {
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
      expect(tracksByPeerIdMid).toEqual(expected)
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
        metadataByPeerIdMid: {},
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
        trackIdToPeerIdMid: {
          [track.id]: userId + '::0',
        },
        tracksByPeerIdMid: {
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
        metadataByPeerIdMid: {},
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
        trackIdToPeerIdMid: {
          [track1.id]: userId + '::0',
          [track2.id]: userId + '::1',
        },
        tracksByPeerIdMid: {
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
      store.dispatch(StreamActions.removeTrack({ track: track1 }))
      const { streams } = store.getState()
      const expected: typeof streams = {
        localStreams: {},
        metadataByPeerIdMid: {},
        streamsByUserId: {},
        trackIdToPeerIdMid: {
          [track1.id]: userId + '::0',
        },
        tracksByPeerIdMid: {
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
      store.dispatch(StreamActions.removeTrack({ track: track1 }))
      const { streams } = store.getState()
      const expected: typeof streams = {
        localStreams: {},
        metadataByPeerIdMid: {},
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
        trackIdToPeerIdMid: {
          [track1.id]: userId + '::0',
          [track2.id]: userId + '::1',
        },
        tracksByPeerIdMid: {
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

  describe('metadata', () => {
    const serverId = '__SERVER__'
    const actualStreamId = 'remote-stream-123'

    it('sets metadata', () => {
      store.dispatch(StreamActions.tracksMetadata({
        metadata: [{
          kind: 'video',
          mid: '0',
          streamId: actualStreamId,
          userId,
        }],
        userId: serverId,
      }))
      const metadata = store.getState().streams.metadataByPeerIdMid
      expect(metadata).toEqual({
        [serverId + '::0']: {
          kind: 'video',
          mid: '0',
          streamId: actualStreamId,
          userId,
        },
      } as typeof metadata)
    })

    describe('addTrack', () => {
      it('uses metadata info to set real userId and streamId', () => {
        store.dispatch(StreamActions.tracksMetadata({
          metadata: [{
            kind: 'video',
            mid: '0',
            streamId: actualStreamId,
            userId,
          }],
          userId: serverId,
        }))
        const track = new MediaStreamTrack()
        store.dispatch(StreamActions.addTrack({
          mid: '0',
          streamId: 'stream-123',
          track,
          userId: serverId,
        }))
        const { streams } = store.getState()
        const expected: StreamsState = {
          localStreams: {},
          metadataByPeerIdMid: {
            [serverId + '::0']: {
              kind: 'video',
              mid: '0',
              streamId: actualStreamId,
              userId,
            },
          },
          streamsByUserId: {
            [userId]: {
              userId,
              streams: [{
                stream: jasmine.any(MediaStream) as any,
                streamId: 'remote-stream-123',
                url: jasmine.any(String) as any,
              }],
            },
          },
          trackIdToPeerIdMid: {
            [track.id]: serverId + '::0',
          },
          tracksByPeerIdMid: {
            [serverId + '::0']: {
              track,
              mid: '0',
              association: {
                userId,
                streamId: 'remote-stream-123',
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
    })

    describe('removeTrack', () => {
      it('uses metadata info to remove correct userId / streamId', () => {
        store.dispatch(StreamActions.tracksMetadata({
          metadata: [{
            kind: 'video',
            mid: '0',
            streamId: actualStreamId,
            userId,
          }],
          userId: serverId,
        }))
        const track = new MediaStreamTrack()
        store.dispatch(StreamActions.addTrack({
          mid: '0',
          streamId: 'stream-123',
          track,
          userId: serverId,
        }))
        store.dispatch(StreamActions.removeTrack({ track }))
        const { streams } = store.getState()
        const expected: StreamsState = {
          localStreams: {},
          metadataByPeerIdMid: {
            [serverId + '::0']: {
              kind: 'video',
              mid: '0',
              streamId: actualStreamId,
              userId,
            },
          },
          streamsByUserId: {},
          trackIdToPeerIdMid: {
            [track.id]: serverId + '::0',
          },
          tracksByPeerIdMid: {
            [serverId + '::0']: {
              track,
              mid: '0',
              association: undefined,
            },
          },
        }
        expect(streams).toEqual(expected)
      })
    })

    describe('setMetadata after addTrack', () => {
      const track1 = new MediaStreamTrack()
      const track2 = new MediaStreamTrack()

      beforeEach(() => {
        store.dispatch(StreamActions.addTrack({
          mid: '0',
          streamId: 'stream-123',
          track: track1,
          userId: serverId,
        }))
        store.dispatch(StreamActions.addTrack({
          mid: '1',
          streamId: 'stream-123',
          track: track2,
          userId: serverId,
        }))
      })

      it('reorganizes existing tracks according to metadata', () => {
        store.dispatch(StreamActions.tracksMetadata({
          metadata: [{
            kind: 'video',
            mid: '0',
            streamId: actualStreamId,
            userId,
          }],
          userId: serverId,
        }))
        const { streams } = store.getState()
        const expected: StreamsState = {
          localStreams: {},
          metadataByPeerIdMid: {
            [serverId + '::0']: {
              kind: 'video',
              mid: '0',
              streamId: actualStreamId,
              userId,
            },
          },
          streamsByUserId: {
            [userId]: {
              userId,
              streams: [{
                stream: jasmine.any(MediaStream) as any,
                streamId: 'remote-stream-123',
                url: jasmine.any(String) as any,
              }],
            },
          },
          trackIdToPeerIdMid: {
            [track1.id]: serverId + '::0',
            [track2.id]: serverId + '::1',
          },
          tracksByPeerIdMid: {
            [serverId + '::0']: {
              track: track1,
              mid: '0',
              association: {
                streamId: 'remote-stream-123',
                userId,
              },
            },
            [serverId + '::1']: {
              track: track2,
              mid: '1',
              association: undefined,
            },
          },
        }
        expect(streams).toEqual(expected)
      })
    })
  })
})
