jest.mock('../window.js')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import Toolbar from './Toolbar.js'
import { MediaStream } from '../window.js'

describe('components/Toolbar', () => {

  class ToolbarWrapper extends React.PureComponent {
    static propTypes = Toolbar.propTypes
    constructor () {
      super()
      this.state = {}
    }
    render () {
      return <Toolbar
        chatVisible={this.props.chatVisible}
        onToggleChat={this.props.onToggleChat}
        onSendFile={this.props.onSendFile}
        messages={this.props.messages}
        stream={this.state.stream || this.props.stream}
      />
    }
  }

  let component, node, mediaStream, url, onToggleChat, onSendFile
  function render () {
    mediaStream = new MediaStream()
    onToggleChat = jest.fn()
    onSendFile = jest.fn()
    component = TestUtils.renderIntoDocument(
      <ToolbarWrapper
        chatVisible
        onToggleChat={onToggleChat}
        onSendFile={onSendFile}
        messages={[]}
        stream={{ mediaStream, url }}
      />
    )
    node = ReactDOM.findDOMNode(component)
  }

  beforeEach(() => {
    render()
  })

  describe('handleChatClick', () => {
    it('toggle chat', () => {
      expect(onToggleChat.mock.calls.length).toBe(0)
      const button = node.querySelector('.chat')
      TestUtils.Simulate.click(button)
      expect(onToggleChat.mock.calls.length).toBe(1)
    })
  })

  describe('handleMicClick', () => {
    it('toggle mic', () => {
      const button = node.querySelector('.mute-audio')
      TestUtils.Simulate.click(button)
      expect(button.classList.contains('on')).toBe(true)
    })
  })

  describe('handleCamClick', () => {
    it('toggle cam', () => {
      const button = node.querySelector('.mute-video')
      TestUtils.Simulate.click(button)
      expect(button.classList.contains('on')).toBe(true)
    })
  })

  describe('handleFullscreenClick', () => {
    it('toggle fullscreen', () => {
      const button = node.querySelector('.fullscreen')
      TestUtils.Simulate.click(button)
      expect(button.classList.contains('on')).toBe(false)
    })
  })

  describe('handleHangoutClick', () => {
    it('hangout', () => {
      const button = node.querySelector('.hangup')
      TestUtils.Simulate.click(button)
      expect(window.location.href).toBe('http://localhost/')
    })
  })

  describe('handleSendFile', () => {
    it('triggers input dialog', () => {
      const sendFileButton = node.querySelector('.send-file')
      const click = jest.fn()
      const file = node.querySelector('input[type=file]')
      file.click = click
      TestUtils.Simulate.click(sendFileButton)
      expect(click.mock.calls.length).toBe(1)
    })
  })

  describe('handleSelectFiles', () => {
    it('iterates through files and calls onSendFile for each', () => {
      const file = node.querySelector('input[type=file]')
      const files = [{ name: 'first' }]
      TestUtils.Simulate.change(file, {
        target: {
          files
        }
      })
      expect(onSendFile.mock.calls).toEqual([[ files[0] ]])
    })
  })

})
