jest.mock('../window')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { getDesktopStream, MediaEnumerateAction } from '../actions/MediaActions'
import { removeLocalStream, StreamTypeDesktop } from '../actions/StreamActions'
import { DialState, DIAL_STATE_IN_CALL, MEDIA_ENUMERATE } from '../constants'
import { LocalStream } from '../reducers/streams'
import { MediaStream } from '../window'
import Toolbar, { ToolbarProps } from './Toolbar'
import { Store, createStore } from '../store'
import { Provider } from 'react-redux'

describe('components/Toolbar', () => {

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
  async function render () {
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

  const action: MediaEnumerateAction = {
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
  }
  let store: Store
  beforeEach(async () => {
    store = createStore()
    store.dispatch(action)
    await render()
  })

  describe('handleChatClick', () => {
    it('toggle chat', () => {
      expect(onToggleChat.mock.calls.length).toBe(0)
      const button = node.querySelector('.chat')!
      TestUtils.Simulate.click(button)
      expect(onToggleChat.mock.calls.length).toBe(1)
    })
  })

  describe('mic dropdown', () => {
    const expected = [false, true, {deviceId: 'mic1', name: 'Microphone'}]
    it('switches microphone', () => {
      const button = node.querySelector('.dropdown .audio')!
      const items = button.parentElement!.querySelectorAll('li')
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

  describe('camera dropdown', () => {
    const expected = [
      false,
      {facingMode: 'user'},
      {deviceId: 'cam1', name: 'Camera'},
    ]
    it('switches camera', () => {
      const button = node.querySelector('.dropdown .video')!
      const items = button.parentElement!.querySelectorAll('li')
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
      }
      await render()
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
      await render()
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
      await render()
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
