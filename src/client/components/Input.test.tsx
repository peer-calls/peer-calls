import Input from './Input'
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { Message } from '../actions/PeerActions'

describe('components/Input', () => {

  let node: Element
  let sendMessage: jest.MockedFunction<(message: Message) => void>
  async function render () {
    sendMessage = jest.fn()
    const div = document.createElement('div')
    await new Promise<Input>(resolve => {
      ReactDOM.render(
        <Input
          ref={input => resolve(input!)}
          sendMessage={sendMessage}
        />,
        div,
      )
    })
    node = div.children[0]
  }
  const message = 'test message'

  beforeEach(() => render())

  describe('send message', () => {

    let input: HTMLTextAreaElement
    beforeEach(() => {
      sendMessage.mockClear()
      input = node.querySelector('textarea')!
    })

    describe('handleSubmit', () => {
      it('does nothing when no message', () => {
        TestUtils.Simulate.change(input, {
          target: { value: '' } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(sendMessage.mock.calls)
        .toEqual([])
      })

      it('sends a message', () => {
        TestUtils.Simulate.change(input, {
          target: { value: message } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(input.value).toBe('')
        expect(sendMessage.mock.calls)
        .toEqual([[ { payload: message, type: 'text' } ]])
      })

      it('sends a nickname command', () => {
        TestUtils.Simulate.change(input, {
          target: { value: '/nick john' } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(sendMessage.mock.calls)
        .toEqual([[ { payload: {nickname: 'john'}, type: 'nickname' } ]])
      })

      it('does not fail when command is empty', () => {
        TestUtils.Simulate.change(input, {
          target: { value: '/nick ' } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(sendMessage.mock.calls)
        .toEqual([[ { payload: {nickname: ''}, type: 'nickname' } ]])
      })

      it('sends message when command is invalid', () => {
        TestUtils.Simulate.change(input, {
          target: { value: '/nick' } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(sendMessage.mock.calls)
        .toEqual([[ { payload: '/nick', type: 'text' } ]])
      })
    })

    describe('handleKeyPress', () => {
      it('sends a message', () => {
        TestUtils.Simulate.change(input, {
          target: { value: message } as any,
        })
        TestUtils.Simulate.keyPress(input, {
          key: 'Enter',
        })
        expect(input.value).toBe('')
        expect(sendMessage.mock.calls)
        .toEqual([[ { payload: message, type: 'text' } ]])
      })

      it('does nothing when other key pressed', () => {
        TestUtils.Simulate.keyPress(input, {
          key: 'test',
        })
        expect(sendMessage.mock.calls.length).toBe(0)
      })
    })

    describe('handleSmileClick', () => {
      it('adds smile to message', () => {
        TestUtils.Simulate.change(input, {
          target: { value: message } as any,
        })
        const div = node.querySelector('.chat-controls-buttons-smile')!
        TestUtils.Simulate.click(div)
        expect(input.value).toBe('test message😑')
      })
    })

  })

})
