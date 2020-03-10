jest.mock('../window')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { getDesktopStream } from '../actions/MediaActions'
import { AddStreamPayload, removeStream } from '../actions/StreamActions'
import { STREAM_TYPE_CAMERA, STREAM_TYPE_DESKTOP } from '../constants'
import { StreamWithURL } from '../reducers/streams'
import { MediaStream } from '../window'
import Toolbar, { ToolbarProps } from './Toolbar'

describe('components/Toolbar', () => {

  interface StreamState {
    stream: AddStreamPayload | null
  }

  class ToolbarWrapper extends React.PureComponent<ToolbarProps, StreamState> {
    state = {
      stream: null,
      desktopStream: null,
    }
    render () {
      return <Toolbar
        chatVisible={this.props.chatVisible}
        onToggleChat={this.props.onToggleChat}
        onHangup={this.props.onHangup}
        onGetDesktopStream={this.props.onGetDesktopStream}
        onRemoveStream={this.props.onRemoveStream}
        onSendFile={this.props.onSendFile}
        messagesCount={this.props.messagesCount}
        stream={this.state.stream || this.props.stream}
        desktopStream={this.state.desktopStream || this.props.desktopStream}
      />
    }
  }

  let node: Element
  let mediaStream: MediaStream
  let url: string
  let onToggleChat: jest.Mock<() => void>
  let onSendFile: jest.Mock<(file: File) => void>
  let onHangup: jest.Mock<() => void>
  let onGetDesktopStream: jest.MockedFunction<typeof getDesktopStream>
  let onRemoveStream: jest.MockedFunction<typeof removeStream>
  let desktopStream: StreamWithURL | undefined
  async function render () {
    mediaStream = new MediaStream()
    onToggleChat = jest.fn()
    onSendFile = jest.fn()
    onHangup = jest.fn()
    onGetDesktopStream = jest.fn().mockImplementation(() => Promise.resolve())
    onRemoveStream = jest.fn()
    const div = document.createElement('div')
    const stream: StreamWithURL = {
      stream: mediaStream,
      type: STREAM_TYPE_CAMERA,
      url,
    }
    await new Promise<ToolbarWrapper>(resolve => {
      ReactDOM.render(
        <ToolbarWrapper
          ref={instance => resolve(instance!)}
          chatVisible
          onHangup={onHangup}
          onToggleChat={onToggleChat}
          onSendFile={onSendFile}
          messagesCount={1}
          stream={stream}
          desktopStream={desktopStream}
          onGetDesktopStream={onGetDesktopStream}
          onRemoveStream={onRemoveStream}
        />,
        div,
      )
    })
    node = div.children[0]
  }

  beforeEach(async () => {
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

  describe('handleMicClick', () => {
    it('toggle mic', () => {
      const button = node.querySelector('.mute-audio')!
      TestUtils.Simulate.click(button)
      expect(button.classList.contains('on')).toBe(true)
    })
  })

  describe('handleCamClick', () => {
    it('toggle cam', () => {
      const button = node.querySelector('.mute-video')!
      TestUtils.Simulate.click(button)
      expect(button.classList.contains('on')).toBe(true)
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

  describe('handleSendFile', () => {
    it('triggers input dialog', () => {
      const sendFileButton = node.querySelector('.send-file')!
      const click = jest.fn()
      const file = node.querySelector('input[type=file]')!;
      (file as any).click = click
      TestUtils.Simulate.click(sendFileButton)
      expect(click.mock.calls.length).toBe(1)
    })
  })

  describe('handleSelectFiles', () => {
    it('iterates through files and calls onSendFile for each', () => {
      const file = node.querySelector('input[type=file]')!
      const files = [{ name: 'first' }] as any
      TestUtils.Simulate.change(file, {
        target: {
          files,
        } as any,
      })
      expect(onSendFile.mock.calls).toEqual([[ files[0] ]])
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
      desktopStream = {
        stream: new MediaStream(),
        type: STREAM_TYPE_DESKTOP,
      }
      await render()
      const shareDesktop = node.querySelector('.stream-desktop')!
      expect(shareDesktop).toBeDefined()
      TestUtils.Simulate.click(shareDesktop)
      await Promise.resolve()
      expect(onRemoveStream.mock.calls.length).toBe(1)
    })
  })

})
