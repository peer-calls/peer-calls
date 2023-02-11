jest.mock('simple-peer')
jest.mock('../insertable-streams')
jest.mock('../socket')
jest.mock('../window')
jest.useFakeTimers()

import SimplePeer from 'simple-peer'
import { dial, hangUp } from '../actions/CallActions'
import * as MediaActions from '../actions/MediaActions'
import { StreamTypeCamera, StreamTypeDesktop } from '../actions/StreamActions'
import { DIAL_STATE_DIALLING, DIAL_STATE_HUNG_UP, DIAL_STATE_IN_CALL, HANG_UP, MEDIA_ENUMERATE, MEDIA_STREAM, PEER_ADD, SOCKET_EVENT_HANG_UP } from '../constants'
import socket from '../socket'
import { createStore, Store } from '../store'
import { MediaStream, MediaStreamTrack } from '../window'
import { MediaConstraint } from './media'


describe('media', () => {

  const nickname = 'john'

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
          // duplicate device should be filtered out.
          // sometimes cameras have two devices with different label (e.g. IR)
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
      await store.dispatch(MediaActions.enumerateDevices({getUserMedia: true}))
      expect(store.getState().media.devices).toEqual({
        audio: [{
          id: 'abcdef2',
          name: 'Audio Input',
          type: 'audioinput',
        }],
        video: [{
          id: 'abcdef1',
          name: 'Video Input',
          type: 'videoinput',
        }],
      })
    })

    it('handles errors', async () => {
      delete (
        navigator.mediaDevices as {enumerateDevices?: unknown}
      ).enumerateDevices
      try {
        await store.dispatch(MediaActions.enumerateDevices({
          getUserMedia: true,
        }))
      } catch (err) {
        // do nothing
      }
      expect(store.getState().media.devices).toEqual({ audio: [], video: [] })
      expect(store.getState().media.error).toBeTruthy()
    })
  })

  describe('media constraints: video', () => {
    type Action = MediaActions.MediaDeviceToggleAction |
      MediaActions.MediaSizeConstraintAction |
      MediaActions.MediaDeviceIdAction

    const tests: {
      name: string
      action: Action
      wantState: MediaConstraint
    }[] = [
      {
        name: 'disable video',
        action: MediaActions.toggleDevice({ kind: 'video', enabled: false }),
        wantState: {
          enabled: false,
          constraints: { facingMode: 'user' },
        },
      },
      {
        name: 'set deviceId',
        action: MediaActions.setDeviceId({ kind: 'video', deviceId: 'abcd' }),
        wantState: {
          enabled: true,
          constraints: { deviceId: 'abcd' },
        },
      },
      {
        name: 'set size constraint',
        action: MediaActions.setSizeConstraint({ width: 640, height: 480 }),
        wantState: {
          enabled: true,
          constraints: { deviceId: 'abcd', width: 640, height: 480 },
        },
      },
      {
        name: 'disable video',
        action: MediaActions.toggleDevice({ kind: 'video', enabled: false }),
        wantState: {
          enabled: false,
          constraints: { deviceId: 'abcd', width: 640, height: 480 },
        },
      },
      {
        name: 'set default deviceId',
        action: MediaActions.setDeviceId({ kind: 'video', deviceId: '' }),
        wantState: {
          enabled: true,
          constraints: { facingMode: 'user', width: 640, height: 480 },
        },
      },
      {
        name: 'set deviceId again',
        action: MediaActions.setDeviceId({ kind: 'video', deviceId: 'efgh' }),
        wantState: {
          enabled: true,
          constraints: { deviceId: 'efgh', width: 640, height: 480 },
        },
      },
      {
        name: 'unset size constraint',
        action: MediaActions.setSizeConstraint(null),
        wantState: {
          enabled: true,
          constraints: { deviceId: 'efgh' },
        },
      },
    ]

    it('test', () => {
      tests.forEach(test => {
        store.dispatch(test.action)
        expect(store.getState().media.video).toEqual(test.wantState)
      })
    })
  })

  describe('media constraints: audio', () => {
    type Action = MediaActions.MediaDeviceToggleAction |
      MediaActions.MediaDeviceIdAction

    const tests: {
      name: string
      action: Action
      wantState: MediaConstraint
    }[] = [
      {
        name: 'disable audio',
        action: MediaActions.toggleDevice({ kind: 'audio', enabled: false }),
        wantState: {
          enabled: false,
          constraints: {},
        },
      },
      {
        name: 'set deviceId',
        action: MediaActions.setDeviceId({ kind: 'audio', deviceId: 'abcd' }),
        wantState: {
          enabled: true,
          constraints: { deviceId: 'abcd' },
        },
      },
      {
        name: 'disable audio',
        action: MediaActions.toggleDevice({ kind: 'audio', enabled: false }),
        wantState: {
          enabled: false,
          constraints: { deviceId: 'abcd' },
        },
      },
      {
        name: 'set default deviceId',
        action: MediaActions.setDeviceId({ kind: 'audio', deviceId: '' }),
        wantState: {
          enabled: true,
          constraints: {},
        },
      },
      {
        name: 'set deviceId again',
        action: MediaActions.setDeviceId({ kind: 'audio', deviceId: 'efgh' }),
        wantState: {
          enabled: true,
          constraints: { deviceId: 'efgh' },
        },
      },
    ]

    it('test', () => {
      tests.forEach(test => {
        store.dispatch(test.action)
        expect(store.getState().media.audio).toEqual(test.wantState)
      })
    })
  })

  describe(MEDIA_STREAM, () => {
    const track = new MediaStreamTrack()
    const stream = new MediaStream()
    stream.addTrack(track)
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
        expect(result.type).toBe(StreamTypeCamera)
      }

      describe('reducers/streams', () => {
        it('adds the local stream to the map of videos', async () => {
          expect(store.getState().streams.localStreams).toEqual({})
          await dispatch()
          const { localStreams } = store.getState().streams
          expect(Object.keys(localStreams).length).toBe(1)
          const s = localStreams[StreamTypeCamera]!
          expect(s).toBeTruthy()
          expect(s.stream).toBe(stream)
          expect(s.streamId).toBe(stream.id)
          expect(s.type).toBe(StreamTypeCamera)
        })
      })

      describe('reducers/peers', () => {
        const peer1 = new SimplePeer()
        const peer2 = new SimplePeer()
        const peers = [peer1, peer2]

        beforeEach(() => {
          store.dispatch({
            type: HANG_UP,
          })
          store.dispatch({
            type: PEER_ADD,
            payload: {
              peerId: '1',
              peer: peer1,
            },
          })
          store.dispatch({
            type: PEER_ADD,
            payload: {
              peerId: '2',
              peer: peer2,
            },
          })
        })

        afterEach(() => {
          store.dispatch({
            type: HANG_UP,
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
        })
      })
    })
  })

  describe('getDesktopStream (getDisplayMedia)', () => {
    const stream: MediaStream = new MediaStream()
    beforeEach(() => {
      (navigator.mediaDevices as any).getDisplayMedia = async () => stream
    })
    async function dispatch() {
      const result = await store.dispatch(MediaActions.getDesktopStream())
      expect(result.stream).toBe(stream)
      expect(result.type).toBe(StreamTypeDesktop)
    }
    it('adds the local stream to the map of videos', async () => {
      expect(store.getState().streams.localStreams).toEqual({})
      await dispatch()
      const {localStreams } = store.getState().streams
      expect(Object.keys(localStreams).length).toBe(1)
      const s = localStreams[StreamTypeDesktop]!
      expect(s.type).toBe(StreamTypeDesktop)
      expect(s.stream).toBe(stream)
      expect(s.streamId).toBe(stream.id)
    })
  })

  describe('dialState', () => {
    async function successfulDial() {
      const promise = store.dispatch(dial({ nickname }))
      expect(store.getState().media.dialState).toBe(DIAL_STATE_DIALLING)
      socket.emit('users', {
        initiator: 'test',
        peerIds: [],
        nicknames: {},
      })
      jest.runAllTimers()
      await promise
      expect(store.getState().media.dialState).toBe(DIAL_STATE_IN_CALL)
    }

    it('has dialState HUNG_UP by default', () => {
      expect(store.getState().media.dialState).toBe(DIAL_STATE_HUNG_UP)
    })
    it('changes state from DIALLING to HUNG_UP', async () => {
      const promise = store.dispatch(dial({ nickname }))
      expect(store.getState().media.dialState).toBe(DIAL_STATE_DIALLING)
      jest.runAllTimers()
      let err!: Error
      try {
        await promise
      } catch (e) {
        err = e
      }
      expect(err).toBeTruthy()
      expect(err.message).toMatch(/Dial timed out/)
      expect(store.getState().media.dialState).toBe(DIAL_STATE_HUNG_UP)
    })
    it('changes state from DIALLING to IN_CALL', async () => {
      await successfulDial()
    })

    it('changes state to HUNG_UP when hangUp action is dispatched', async() => {
      await successfulDial()
      const promise = new Promise<void>(
        resolve => socket.once(SOCKET_EVENT_HANG_UP, () => resolve()),
      )
      store.dispatch(hangUp())
      expect(store.getState().media.dialState).toBe(DIAL_STATE_HUNG_UP)
      await promise
    })
  })

})
