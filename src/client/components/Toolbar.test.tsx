jest.mock('simple-peer')
jest.mock('../window')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { Provider } from 'react-redux'
import { applyMiddleware, createStore } from 'redux'
import SimplePeer from 'simple-peer'
import { getDesktopStream, MediaKind, setAudioConstraint, setVideoConstraint } from '../actions/MediaActions'
import { removeLocalStream, StreamTypeCamera, StreamTypeDesktop } from '../actions/StreamActions'
import { DialState, DIAL_STATE_IN_CALL, MEDIA_ENUMERATE, MEDIA_STREAM, MEDIA_TRACK, MEDIA_TRACK_ENABLE, PEER_ADD } from '../constants'
import reducers from '../reducers'
import { LocalStream } from '../reducers/streams'
import { middlewares, Store } from '../store'
import { MediaStream, MediaStreamTrack } from '../window'
import Toolbar, { ToolbarProps } from './Toolbar'
import { deferred } from '../deferred'

interface StreamState {
  cameraStream: LocalStream | null
  desktopStream: LocalStream | null
}

class ToolbarWrapper extends React.PureComponent<ToolbarProps, StreamState> {
  state = {
    cameraStream: null,
    desktopStream: null,
  }
  render () {
    return <Toolbar
      chatVisible={this.props.chatVisible}
      dialState={this.props.dialState}
      nickname={this.props.nickname}
      onToggleChat={this.props.onToggleChat}
      onHangup={this.props.onHangup}
      onGetDesktopStream={this.props.onGetDesktopStream}
      onRemoveLocalStream={this.props.onRemoveLocalStream}
      messagesCount={this.props.messagesCount}
      desktopStream={this.state.desktopStream || this.props.desktopStream}
    />
  }
}

let node: Element
let onToggleChat: jest.Mock<() => void>
let onHangup: jest.Mock<() => void>
let onGetDesktopStream: jest.MockedFunction<typeof getDesktopStream>
let onRemoveLocalStream: jest.MockedFunction<typeof removeLocalStream>
let desktopStream: LocalStream | undefined
let dialState: DialState
const nickname = 'john'
async function render (store: Store) {
  dialState = DIAL_STATE_IN_CALL
  onToggleChat = jest.fn()
  onHangup = jest.fn()
  onGetDesktopStream = jest.fn().mockImplementation(() => Promise.resolve())
  onRemoveLocalStream = jest.fn()
  const div = document.createElement('div')
  await new Promise<ToolbarWrapper>(resolve => {
    ReactDOM.render(
      <Provider store={store}>
        <ToolbarWrapper
          ref={instance => resolve(instance!)}
          dialState={dialState}
          chatVisible
          onHangup={onHangup}
          onToggleChat={onToggleChat}
          messagesCount={1}
          nickname={nickname}
          desktopStream={desktopStream}
          onGetDesktopStream={onGetDesktopStream}
          onRemoveLocalStream={onRemoveLocalStream}
        />
      </Provider>,
      div,
    )
  })
  node = div
}

describe('components/Toolbar', () => {

  let store: Store
  beforeEach(async () => {
    store = createStore(reducers, applyMiddleware(...middlewares))
    await render(store)
  })

  describe('handleChatClick', () => {
    it('toggle chat', () => {
      expect(onToggleChat.mock.calls.length).toBe(0)
      const button = node.querySelector('.chat')!
      TestUtils.Simulate.click(button)
      expect(onToggleChat.mock.calls.length).toBe(1)
    })
  })

  describe('handleFullscreenClick', () => {
    it('toggle fullscreen', () => {
      const button = node.querySelector('.fullscreen')!
      TestUtils.Simulate.click(button)
      expect(button.classList.contains('on')).toBe(false)
    })
  })

  describe('handleHangoutClick', () => {
    it('hangout', () => {
      const button = node.querySelector('.hangup')!
      TestUtils.Simulate.click(button)
      expect(window.location.href).toBe('http://localhost/')
    })
  })

  describe('onHangup', () => {
    it('calls onHangup callback', () => {
      expect(onHangup.mock.calls.length).toBe(0)
      const hangup = node.querySelector('.hangup')!
      expect(hangup).toBeDefined()
      TestUtils.Simulate.click(hangup)
      expect(onHangup.mock.calls.length).toBe(1)
    })
  })

  describe('desktop sharing', () => {
    it('starts desktop sharing', async () => {
      const shareDesktop = node.querySelector('.stream-desktop')!
      expect(shareDesktop).toBeDefined()
      TestUtils.Simulate.click(shareDesktop)
      await Promise.resolve()
      expect(onGetDesktopStream.mock.calls.length).toBe(1)
    })
    it('stops desktop sharing', async () => {
      const stream = new MediaStream()
      desktopStream = {
        stream,
        streamId: stream.id,
        type: StreamTypeDesktop,
        mirror: false,
      }
      await render(store)
      const shareDesktop = node.querySelector('.stream-desktop')!
      expect(shareDesktop).toBeDefined()
      TestUtils.Simulate.click(shareDesktop)
      await Promise.resolve()
      expect(onRemoveLocalStream.mock.calls.length).toBe(1)
    })
  })

  describe('share / copy invitation url', () => {

    let promise: Promise<string>
    beforeEach(() => {
      promise = new Promise<string>(resolve => {
        (navigator.clipboard as any) = {}
        navigator.clipboard.writeText = async text => {
          resolve(text)
        }
      })
    })

    it('copies invite url using navigator.clipboard', async () => {
      await render(store)
      const copyUrl = node.querySelector('.copy-url')!
      expect(copyUrl).toBeDefined()
      TestUtils.Simulate.click(copyUrl)
      const result = await promise
      expect(result).toMatch(/john has invited you/)
    })

    it('opens share dialog when available', async () => {
      let res: (value: any) => void
      const p = new Promise<any>(resolve => res = resolve)
      ;(navigator as any).share = (value: any) => res(value)
      await render(store)
      const copyUrl = node.querySelector('.copy-url')!
      expect(copyUrl).toBeDefined()
      TestUtils.Simulate.click(copyUrl)
      expect(await p).toEqual({
        title: 'Peer Call',
        text: 'john has invited you to a meeting on Peer Calls',
        url: jasmine.stringMatching(/^http/),
      })
    })
  })

})

describe('components/Toolbar track dropdowns', () => {

  let store: Store
  let stream: MediaStream
  let peer: SimplePeer.Instance
  const peerId = 'peer-1'
  let [ promise, resolve ] = deferred<void>()
  beforeEach(async () => {
    peer = new SimplePeer()
    stream = new MediaStream()

    ;[promise, resolve ] = deferred<void>()

    const _reducers: typeof reducers = (state, action) => {
      if (
        action.type === MEDIA_TRACK_ENABLE ||
        action.type === MEDIA_TRACK &&
        (action as any).status === 'resolved'
      ) {
        resolve()
      }
      return reducers(state, action)
    }

    store = createStore(
      _reducers,
      applyMiddleware(...middlewares),
    )
    store.dispatch({
      type: PEER_ADD,
      payload: {
        userId: peerId,
        peer,
      },
    })
    store.dispatch({
      type: MEDIA_ENUMERATE,
      status: 'resolved',
      payload: [{
        id: 'cam1',
        name: 'Camera',
        type: 'videoinput',
      }, {
        id: 'mic1',
        name: 'Microphone',
        type: 'audioinput',
      }],
    })
    await render(store)
  })

  describe('mic and camera dropdowns', () => {
    let audioTrack: MediaStreamTrack
    let videoTrack: MediaStreamTrack
    beforeEach(() => {
      audioTrack = new MediaStreamTrack()
      ;(audioTrack.kind as any) = 'audio'
      videoTrack = new MediaStreamTrack()
      ;(videoTrack.kind as any) = 'video'

      window.navigator.mediaDevices.getUserMedia =
        async (constraints: MediaStreamConstraints) => {
          const stream = new MediaStream()

          if (!constraints.audio && !constraints.video) {
            // mimic browser behavior
            throw new Error('Audio or video must be defined')
          }

          if (constraints.audio) {
            stream.addTrack(audioTrack)
          }
          if (constraints.video) {
            stream.addTrack(videoTrack)
          }

          return stream
        }
    })

    function getDevices(kind: MediaKind): HTMLLIElement[] {
      const button = node.querySelector('.dropdown .' + kind)!
      const items = button.parentElement!.querySelectorAll('li')
      expect(items).toBeDefined()
      return Array.from(items)
    }

    describe('no local stream', () => {
      it('track disable does nothing when no local stream', async () => {
        const devices = getDevices('video')

        // add track
        TestUtils.Simulate.click(devices[1])
        await promise
        ;[promise, resolve] = deferred<void>()

        // disable track
        TestUtils.Simulate.click(devices[0])
        await promise
        ;[promise, resolve] = deferred<void>()

        // enable
        TestUtils.Simulate.click(devices[1])
        await promise
      })
    })

    describe('existing camera stream', () => {

      beforeEach(() => {
        store.dispatch({
          type: MEDIA_STREAM,
          payload: {
            stream,
            type: StreamTypeCamera,
          },
          status: 'resolved',
        })
      })

      describe('no old track => new track', () => {
        beforeEach(() => {
          store.dispatch(setVideoConstraint(false))
          store.dispatch(setAudioConstraint(false))
        })
        it('adds a track to existing peer stream', async () => {
          const device = getDevices('video')[2]
          TestUtils.Simulate.click(device)
          await promise
          const addTrack = store.getState().peers[peerId].addTrack as jest.Mock
          expect(addTrack.mock.calls).toEqual([[ videoTrack, stream ]])
        })
      })

      describe('old track => ', () => {
        let oldTrack: MediaStreamTrack
        let devices: HTMLLIElement[]
        beforeEach(async () => {
          store.dispatch(setVideoConstraint(false))
          store.dispatch(setAudioConstraint(false))
          devices = getDevices('video')
          TestUtils.Simulate.click(devices[1])
          await promise
          ;[promise, resolve] = deferred<void>()
          oldTrack = videoTrack
          videoTrack = new MediaStreamTrack()
          ;(videoTrack as any).kind = 'video'
        })
        describe('new track', () => {
          it('replaces peer track with new track in same stream', async () => {
            TestUtils.Simulate.click(devices[2])
            await promise
            const replaceTrack =
              store.getState().peers[peerId].replaceTrack as jest.Mock
            expect(replaceTrack.mock.calls)
            .toEqual([[ oldTrack, videoTrack, stream ]])
          })
        })

        async function disableTrack() {
          expect(oldTrack.enabled).toBe(true)
          TestUtils.Simulate.click(devices[0])
          await promise
          expect(oldTrack.enabled).toBe(false)
          expect(stream.getTracks()).toEqual([ oldTrack ])
        }

        describe('no new track (mute) and unmute', () => {
          it('disables existing track when no new track', async () => {
            await disableTrack()
          })
        })

        describe('enable (unmute)', () => {
          beforeEach(async () => {
            await disableTrack()
            ;[promise, resolve] = deferred<void>()
          })
          it('enables existing track when previous track clicked', async () => {
            expect(oldTrack.enabled).toBe(false)
            TestUtils.Simulate.click(devices[1])
            await promise
            expect(oldTrack.enabled).toBe(true)
            expect(stream.getTracks()).toEqual([ oldTrack ])
          })
        })

      })

      describe('mic', () => {
        const expected = [false, true, {deviceId: 'mic1'}]
        it('switches microphone', () => {
          const button = node.querySelector('.dropdown .audio')!
          const items = getDevices('audio')
          expect(items.length).toBe(3)
          items.forEach((item, i) => {
            expect(button).toBeTruthy()
            TestUtils.Simulate.click(button)
            TestUtils.Simulate.click(item)
            expect(store.getState().media.audio).toEqual(expected[i])
            // TODO test for getMediaStream
          })
        })
      })

      describe('camera', () => {
        const expected = [
          false,
          {facingMode: 'user'},
          {deviceId: 'cam1'},
        ]
        it('switches camera', () => {
          const button = node.querySelector('.dropdown .video')!
          const items = getDevices('video')
          expect(items.length).toBe(3)
          items.forEach((item, i) => {
            expect(button).toBeTruthy()
            TestUtils.Simulate.click(button)
            TestUtils.Simulate.click(item)
            expect(store.getState().media.video).toEqual(expected[i])
            // TODO test for getMediaStream
          })
        })
      })

    })

  })

})
