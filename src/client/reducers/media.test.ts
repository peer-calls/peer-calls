jest.mock('simple-peer')

import * as MediaActions from '../actions/MediaActions'
import { MEDIA_ENUMERATE, MEDIA_VIDEO_CONSTRAINT_SET, MEDIA_AUDIO_CONSTRAINT_SET, MEDIA_STREAM, ME, PEERS_DESTROY, PEER_ADD, STREAM_TYPE_CAMERA, STREAM_TYPE_DESKTOP } from '../constants'
import { createStore, Store } from '../store'
import SimplePeer from 'simple-peer'

describe('media', () => {

  let store: Store
  beforeEach(() => {
    store = createStore();
    (navigator as any).mediaDevices = {}
  })

  afterEach(() => {
    delete (navigator as any).mediaDevices
  })

  function toJSON(this: MediaDeviceInfo) {
    return JSON.stringify(this)
  }

  describe(MEDIA_ENUMERATE, () => {
    beforeEach(() => {
      navigator.mediaDevices.enumerateDevices = async () => {
        const result: MediaDeviceInfo[] = [{
          deviceId: 'abcdef1',
          groupId: 'group1',
          kind: 'videoinput',
          label: 'Video Input',
          toJSON,
        }, {
          deviceId: 'abcdef2',
          groupId: 'group1',
          kind: 'audioinput',
          label: 'Audio Input',
          toJSON,
        }, {
          deviceId: 'abcdef3',
          groupId: 'group2',
          kind: 'audiooutput',
          label: 'Audio Output',
          toJSON,
        }]
        return result
      }
    })

    it('retrieves a list of audioinput/videoinput devices', async () => {
      await store.dispatch(MediaActions.enumerateDevices())
      expect(store.getState().media.devices).toEqual([{
        id: 'abcdef1',
        name: 'Video Input',
        type: 'videoinput',
      }, {
        id: 'abcdef2',
        name: 'Audio Input',
        type: 'audioinput',
      }])
    })

    it('handles errors', async () => {
      delete navigator.mediaDevices.enumerateDevices
      try {
        await store.dispatch(MediaActions.enumerateDevices())
      } catch (err) {
        // do nothing
      }
      expect(store.getState().media.devices).toEqual([])
      expect(store.getState().media.error).toBeTruthy()
    })
  })

  describe(MEDIA_VIDEO_CONSTRAINT_SET, () => {
    it('sets constraints for video device', () => {
      expect(store.getState().media.video).toEqual({ facingMode: 'user' })
      const constraint: MediaActions.VideoConstraint = true
      store.dispatch(MediaActions.setVideoConstraint(constraint))
      expect(store.getState().media.video).toEqual(constraint)
    })
  })

  describe(MEDIA_AUDIO_CONSTRAINT_SET, () => {
    it('sets constraints for audio device', () => {
      expect(store.getState().media.audio).toBe(true)
      const constraint: MediaActions.AudioConstraint = { deviceId: 'abcd' }
      store.dispatch(MediaActions.setAudioConstraint(constraint))
      expect(store.getState().media.audio).toEqual(constraint)
    })
  })

  describe(MEDIA_STREAM, () => {
    const track: MediaStreamTrack = {} as unknown as MediaStreamTrack
    const stream: MediaStream = {
      getTracks() {
        return [track]
      },
    } as MediaStream
    describe('using navigator.mediaDevices.getUserMedia', () => {

      beforeEach(() => {
        navigator.mediaDevices.getUserMedia = async () => stream
      })

      async function dispatch() {
        const result = await store.dispatch(MediaActions.getMediaStream({
          audio: true,
          video: true,
        }))
        expect(result.stream).toBe(stream)
        expect(result.type).toBe(STREAM_TYPE_CAMERA)
        expect(result.userId).toBe(ME)
      }

      describe('reducers/streams', () => {
        it('adds the local stream to the map of videos', async () => {
          expect(store.getState().streams[ME]).toBeFalsy()
          await dispatch()
          const localStreams = store.getState().streams[ME]
          expect(localStreams).toBeTruthy()
          expect(localStreams.streams.length).toBe(1)
          expect(localStreams.streams[0].type).toBe(STREAM_TYPE_CAMERA)
          expect(localStreams.streams[0].stream).toBe(stream)
        })
      })

      describe('reducers/peers', () => {
        const peer1 = new SimplePeer()
        const peer2 = new SimplePeer()
        const peers = [peer1, peer2]

        beforeEach(() => {
          store.dispatch({
            type: PEERS_DESTROY,
          })
          store.dispatch({
            type: PEER_ADD,
            payload: {
              userId: '1',
              peer: peer1,
            },
          })
          store.dispatch({
            type: PEER_ADD,
            payload: {
              userId: '2',
              peer: peer2,
            },
          })
        })

        afterEach(() => {
          store.dispatch({
            type: PEERS_DESTROY,
          })
        })

        it('adds local camera stream to all peers', async () => {
          await dispatch()
          peers.forEach(peer => {
            expect((peer.addTrack as jest.Mock).mock.calls)
            .toEqual([[ track, stream ]])
            expect((peer.removeTrack as any).mock.calls).toEqual([])
          })
          await dispatch()
          peers.forEach(peer => {
            expect((peer.addTrack as jest.Mock).mock.calls)
            .toEqual([[ track, stream ], [ track, stream ]])
            expect((peer.removeTrack as jest.Mock).mock.calls)
            .toEqual([[ track, stream ]])
          })
        })
      })
    });

    ['getUserMedia', 'mozGetUserMedia', 'webkitGetUserMedia'].forEach(item => {
      describe('compatibility: navigator.' + item, () => {
        beforeEach(() => {
          const getUserMedia: typeof navigator.getUserMedia =
            (constraint, onSuccess, onError) => onSuccess(stream);
          (navigator as any)[item] = getUserMedia
        })
        afterEach(() => {
          delete (navigator as any)[item]
        })
        it('returns a promise with media stream' + item, async () => {
          const promise = MediaActions.getMediaStream({
            audio: true,
            video: true,
          })
          expect(promise.type).toBe('MEDIA_STREAM')
          expect(promise.status).toBe('pending')
          const result = await promise
          expect(result.stream).toBe(stream)
          expect(result.userId).toBe(ME)
        })
      })
    })
  })

  describe('getDesktopStream (getDisplayMedia)', () => {
    const stream: MediaStream = {} as MediaStream
    beforeEach(() => {
      (navigator.mediaDevices as any).getDisplayMedia = async () => stream
    })
    async function dispatch() {
      const result = await store.dispatch(MediaActions.getDesktopStream())
      expect(result.stream).toBe(stream)
      expect(result.type).toBe(STREAM_TYPE_DESKTOP)
      expect(result.userId).toBe(ME)
    }
    it('adds the local stream to the map of videos', async () => {
      expect(store.getState().streams[ME]).toBeFalsy()
      await dispatch()
      const localStreams = store.getState().streams[ME]
      expect(localStreams).toBeTruthy()
      expect(localStreams.streams.length).toBe(1)
      expect(localStreams.streams[0].type).toBe(STREAM_TYPE_DESKTOP)
      expect(localStreams.streams[0].stream).toBe(stream)
    })
  })

})
