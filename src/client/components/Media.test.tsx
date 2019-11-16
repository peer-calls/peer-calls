import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { Provider } from 'react-redux'
import { createStore, Store } from '../store'
import { Media } from './Media'
import { MEDIA_ENUMERATE } from '../constants'

describe('Media', () => {

  const onSave = jest.fn()
  beforeEach(() => {
    jest.resetAllMocks()
    store = createStore()
    store.dispatch({
      type: MEDIA_ENUMERATE,
      status: 'resolved',
      payload: [{
        id: '123',
        name: 'Audio Input',
        type: 'audioinput',
      }, {
        id: '456',
        label: 'Video Input',
        name: 'videoinput',
      }],
    })
  })

  let store: Store
  async function render() {
    const div = document.createElement('div')
    const node = await new Promise<HTMLDivElement>(resolve => {
      ReactDOM.render(
        <div ref={div => resolve(div!)}>
          <Provider store={store}>
            <Media onSave={onSave} />
          </Provider>
        </div>,
        div,
      )
    })
    return node.children[0]
  }

  describe('submit', () => {
    it('calls onSave', async () => {
      const node = await render()
      expect(node.tagName).toBe('FORM')
      TestUtils.Simulate.submit(node)
      expect(onSave.mock.calls.length).toBe(1)
    })
  })

  describe('onVideoChange', () => {
    it('calls onSetVideoConstraint', async () => {
      const node = await render()
      const select = node.querySelector('select.media-video')!
      TestUtils.Simulate.change(select, {
        target: {
          value: '{"deviceId":123}',
        } as any,
      })
      expect(store.getState().media.video).toEqual({ deviceId: 123 })
    })
  })

  describe('onAudioChange', () => {
    it('calls onSetAudioConstraint', async () => {
      const node = await render()
      const select = node.querySelector('select.media-audio')!
      TestUtils.Simulate.change(select, {
        target: {
          value: '{"deviceId":456}',
        } as any,
      })
      expect(store.getState().media.audio).toEqual({ deviceId: 456 })
    })
  })

})
