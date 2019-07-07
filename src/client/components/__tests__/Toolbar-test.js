jest.mock('../../window.js')
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import Toolbar from '../Toolbar.js'
import { MediaStream } from '../../window.js'

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
        messages={this.props.messages}
        stream={this.state.stream || this.props.stream}
      />
    }
  }

  let component, node, mediaStream, url, onToggleChat
  function render () {
    mediaStream = new MediaStream()
    onToggleChat = jest.fn()
    component = TestUtils.renderIntoDocument(
      <ToolbarWrapper
        chatVisible
        onToggleChat={onToggleChat}
        messages={[]}
        stream={{ mediaStream, url }}
      />
    )
    node = ReactDOM.findDOMNode(component)
  }

  describe('render', () => {
    it('should not fail', () => {
      render()
    })
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

})
