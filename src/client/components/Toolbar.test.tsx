jest.mock('../window')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import Toolbar, { ToolbarProps } from './Toolbar'
import { MediaStream } from '../window'
import { AddStreamPayload } from '../actions/StreamActions'

describe('components/Toolbar', () => {

  interface StreamState {
    stream: AddStreamPayload | null
  }

  class ToolbarWrapper extends React.PureComponent<ToolbarProps, StreamState> {
    state = {
      stream: null,
    }
    render () {
      return <Toolbar
        chatVisible={this.props.chatVisible}
        onToggleChat={this.props.onToggleChat}
        onHangup={this.props.onHangup}
        onSendFile={this.props.onSendFile}
        messagesCount={this.props.messagesCount}
        stream={this.state.stream || this.props.stream}
      />
    }
  }

  let node: Element
  let mediaStream: MediaStream
  let url: string
  let onToggleChat: jest.Mock<() => void>
  let onSendFile: jest.Mock<(file: File) => void>
  let onHangup: jest.Mock<() => void>
  async function render () {
    mediaStream = new MediaStream()
    onToggleChat = jest.fn()
    onSendFile = jest.fn()
    onHangup = jest.fn()
    const div = document.createElement('div')
    await new Promise<ToolbarWrapper>(resolve => {
      ReactDOM.render(
        <ToolbarWrapper
          ref={instance => resolve(instance!)}
          chatVisible
          onHangup={onHangup}
          onToggleChat={onToggleChat}
          onSendFile={onSendFile}
          messagesCount={1}
          stream={{ userId: '', stream: mediaStream, url }}
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

})
