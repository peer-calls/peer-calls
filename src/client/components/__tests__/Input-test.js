import Input from '../Input.js'
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'

describe('components/Input', () => {

  let component, node, notify, sendMessage
  function render () {
    notify = jest.fn()
    sendMessage = jest.fn()
    component = TestUtils.renderIntoDocument(
      <Input
        sendMessage={sendMessage}
        notify={notify}
      />
    )
    node = ReactDOM.findDOMNode(component)
  }
  let message = 'test message'

  beforeEach(() => render())

  describe('send message', () => {

    let input
    beforeEach(() => {
      sendMessage.mockClear()
      input = node.querySelector('input')
      TestUtils.Simulate.change(input, {
        target: { value: message }
      })
      expect(input.value).toBe(message)
    })

    describe('handleSubmit', () => {
      it('sends a message', () => {
        TestUtils.Simulate.submit(node)
        expect(input.value).toBe('')
        expect(sendMessage.mock.calls).toEqual([[ message ]])
        expect(notify.mock.calls).toEqual([[ `You: ${message}` ]])
      })
    })

    describe('handleKeyPress', () => {
      it('sends a message', () => {
        TestUtils.Simulate.keyPress(input, {
          key: 'Enter'
        })
        expect(input.value).toBe('')
        expect(sendMessage.mock.calls).toEqual([[ message ]])
        expect(notify.mock.calls).toEqual([[ `You: ${message}` ]])
      })

      it('does nothing when other key pressed', () => {
        TestUtils.Simulate.keyPress(input, {
          key: 'test'
        })
        expect(sendMessage.mock.calls.length).toBe(0)
        expect(notify.mock.calls.length).toBe(0)
      })
    })

  })

})
