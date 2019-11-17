jest.mock('../window')

import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { Provider } from 'react-redux'
import { createStore, Store } from '../store'
import { Media } from './Media'
import { MEDIA_ENUMERATE } from '../constants'

describe('Media', () => {

  beforeEach(() => {
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
            <Media />
          </Provider>
        </div>,
        div,
      )
    })
    return node.children[0]
  }

  describe('submit', () => {
    const stream = {} as MediaStream
    let promise: Promise<MediaStream>
    beforeEach(() => {
      navigator.mediaDevices.getUserMedia = async () => {
        promise = Promise.resolve(stream)
        return promise
      }
    })
    it('tries to retrieve audio/video media stream', async () => {
      const node = (await render()).querySelector('.media')!
      expect(node.tagName).toBe('FORM')
      TestUtils.Simulate.submit(node)
      expect(promise).toBeDefined()
      await promise
    })
  })

  describe('onVideoChange', () => {
    it('calls onSetVideoConstraint', async () => {
      const node = await render()
      const select = node.querySelector('select[name=video-input]')!
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
      const select = node.querySelector('select[name="audio-input"]')!
      TestUtils.Simulate.change(select, {
        target: {
          value: '{"deviceId":456}',
        } as any,
      })
      expect(store.getState().media.audio).toEqual({ deviceId: 456 })
    })
  })

})
