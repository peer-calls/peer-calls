import Input from './Input'
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'

describe('components/Input', () => {

  let node: Element
  let sendText: jest.MockedFunction<(message: string) => void>
  let sendFile: jest.MockedFunction<(file: File) => void>
  async function render () {
    sendText = jest.fn()
    sendFile = jest.fn()
    const div = document.createElement('div')
    await new Promise<Input>(resolve => {
      ReactDOM.render(
        <Input
          ref={input => resolve(input!)}
          sendFile={sendFile}
          sendText={sendText}
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
      sendText.mockClear()
      input = node.querySelector('textarea')!
    })

    describe('handleSubmit', () => {
      it('does nothing when no message', () => {
        TestUtils.Simulate.change(input, {
          target: { value: '' } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(sendText.mock.calls)
        .toEqual([])
      })

      it('sends a message', () => {
        TestUtils.Simulate.change(input, {
          target: { value: message } as any,
        })
        TestUtils.Simulate.submit(node)
        expect(input.value).toBe('')
        expect(sendText.mock.calls)
        .toEqual([[ message ]])
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
        expect(sendText.mock.calls)
        .toEqual([[ message ]])
      })

      it('does nothing when other key pressed', () => {
        TestUtils.Simulate.keyPress(input, {
          key: 'test',
        })
        expect(sendText.mock.calls.length).toBe(0)
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

  describe('handleSendFile', () => {
    it('triggers input dialog', () => {
      const sendFileButton = node
      .querySelector('.chat-controls-buttons-send-file')!
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
      expect(sendFile.mock.calls).toEqual([[ files[0] ]])
    })
  })

})
