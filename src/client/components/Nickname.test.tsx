import { Nickname } from './Nickname'
import { NicknameMessage } from '../actions/PeerActions'
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'

describe('Nickname', () => {

  // let component: Nickname
  let nickname: string
  let onChange: jest.MockedFunction<(message: NicknameMessage) => void>
  let localUser: boolean | undefined
  let div: HTMLDivElement

  async function render() {
    nickname = 'john'
    onChange = jest.fn()

    div = document.createElement('div')
    await new Promise<Nickname>(resolve => {
      ReactDOM.render(
        <Nickname
          ref={instance => resolve(instance!)}
          onChange={onChange}
          value={nickname}
          localUser={localUser}
        />,
        div,
      )
    })
  }

  afterEach(() => {
    ReactDOM.unmountComponentAtNode(div)
  })

  describe('read-only', () => {
    it('displays static nickname for other users', async () => {
      localUser = undefined
      await render()
      expect(div.children[0].tagName).toBe('SPAN')
    })
  })

  describe('editable', () => {
    let input: HTMLInputElement
    beforeEach(async () => {
      localUser = true
      await render()
      input = div.children[0] as HTMLInputElement
      expect(input.value).toBe('john')
      expect(input.tagName).toBe('INPUT')
      TestUtils.Simulate.change(input, {
        target: { value: 'jack' },
      } as any)
      expect(input.value).toBe('jack')
    })

    it('edits nickname on blur', () => {
      TestUtils.Simulate.blur(input)
      expect(onChange.mock.calls).toEqual([[{
        type: 'nickname',
        payload: {nickname: 'jack'},
      }]])
    })

    describe('keyPress', () => {
      it('blurs (and edits nickname) on enter', () => {
        const blurMock = jest.fn()
        input.blur = blurMock
        TestUtils.Simulate.keyPress(input, {
          key: 'Enter',
        })
        expect(blurMock.mock.calls.length).toBe(1)
      })
      it('does nothing on other keys', () => {
        TestUtils.Simulate.keyPress(input, {
          key: 'a',
        })
        expect(onChange.mock.calls).toEqual([])
      })
    })
  })

})
