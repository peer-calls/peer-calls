jest.mock('../actions/CallActions')
jest.mock('../socket')
jest.mock('../window')
jest.useFakeTimers()

import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import { Provider } from 'react-redux'
import { AnyAction, applyMiddleware, createStore } from 'redux'
import { init } from '../actions/CallActions'
import * as constants from '../constants'
import reducers from '../reducers'
import { middlewares, State, Store } from '../store'
import { MediaStream } from '../window'
import App from './App'
import { StreamTypeCamera } from '../actions/StreamActions'
import media from '../reducers/media'

describe('App', () => {

  const connectedAction = { type: constants.SOCKET_CONNECTED }

  let store: Store
  let state: Partial<State>
  let dispatchSpy: jest.SpyInstance<AnyAction, AnyAction[]>
  beforeEach(() => {
    state = {
      media: media(undefined, {} as any),
    };
    (init as jest.Mock).mockReturnValue(connectedAction)

    window.HTMLMediaElement.prototype.play = jest.fn()
  })

  afterEach(() => {
    if (dispatchSpy) {
      dispatchSpy.mockReset()
      dispatchSpy.mockRestore()
    }
  })

  let node: Element
  async function render () {
    store = createStore(
      reducers,
      state,
      applyMiddleware(...middlewares),
    )
    dispatchSpy = jest.spyOn(store, 'dispatch')
    const div = document.createElement('div')
    node = await new Promise<HTMLDivElement>(resolve => {
      ReactDOM.render(
        <Provider store={store}>
          <div ref={div => resolve(div!)}>
            <App />
          </div>
        </Provider>,
        div,
      )
    })
  }

  describe('render', () => {
    it('renders without issues', async () => {
      await render()
      expect(node).toBeTruthy()
      expect((init as jest.Mock).mock.calls.length).toBe(1)
    })
  })

  describe('chat toggle', () => {
    it('toggles chat state', async () => {
      state.media!.dialState = constants.DIAL_STATE_IN_CALL
      await render()
      const chatButton = node.querySelector('.toolbar .toolbar-btn-chat')!
      expect(chatButton).toBeTruthy()
      TestUtils.Simulate.click(chatButton)
      TestUtils.Simulate.click(chatButton)
    })
  })

  describe('state', () => {
    beforeEach(async () => {
      const localStream = new MediaStream()
      const remoteStream = new MediaStream()
      state.streams = {
        localStreams: {
          [StreamTypeCamera]: {
            stream: localStream,
            streamId: localStream.id,
            type: StreamTypeCamera,
          },
        },
        pubStreams: {},
        pubStreamsKeysByPeerId: {},
        remoteStreamsKeysByClientId: {
          'other-user': {
            [remoteStream.id]: undefined,
          },
        },
        remoteStreams: {
          [remoteStream.id]: {
            stream: remoteStream,
            streamId: remoteStream.id,
            url: 'blob://',
          },
        },
      }
      state.media!.dialState = constants.DIAL_STATE_IN_CALL
      state.peers = {
        'other-user': {
          instance: {} as any,
          senders: {} as any,
        },
      }
      state.nicknames = {
        [constants.ME]: 'local user',
        'other-user': 'remote user',
      }
      state.notifications = {
        'notification1': {
          id: 'notification1',
          message: 'test',
          type: 'warning',
        },
      }
      await render()
    })

    describe('video', () => {
      beforeEach(() => {
        dispatchSpy.mockClear()
      })

      it('forces play on click', () => {
        const video = node.querySelector('.video-container video')!
        expect(video).toBeTruthy()
        TestUtils.Simulate.mouseDown(video)
        TestUtils.Simulate.mouseUp(video)
        TestUtils.Simulate.click(video)
        expect(dispatchSpy.mock.calls[0][0].type).toBe(constants.MEDIA_PLAY)
      })
    })

    describe('settings', () => {
      beforeEach(() => {
        dispatchSpy.mockClear()
      })

      it('modifies the gridKind setting', () => {
        TestUtils.Simulate.click(node.querySelector('.sidebar-menu-settings')!)

        const radioAuto = node.querySelector(
          '.sidebar .settings .settings-grid-kind-auto',
        ) as HTMLInputElement
        const radioLegacy = node.querySelector(
          '.sidebar .settings .settings-grid-kind-legacy',
        ) as HTMLInputElement
        const radioAspect = node.querySelector(
          '.sidebar .settings .settings-grid-kind-aspect',
        ) as HTMLInputElement

        expect(radioAuto).toBeTruthy()
        expect(radioLegacy).toBeTruthy()
        expect(radioAspect).toBeTruthy()

        expect(radioAuto.checked).toBe(true)
        expect(radioLegacy.checked).toBe(false)
        expect(radioAspect.checked).toBe(false)

        let els = node.querySelectorAll('.videos-grid-aspect-ratio')
        expect(els.length).toBe(0)

        els = node.querySelectorAll('.videos-grid-flex')
        expect(els.length).toBe(1)

        TestUtils.Simulate.change(radioLegacy)

        expect(radioAuto.checked).toBe(false)
        expect(radioLegacy.checked).toBe(true)
        expect(radioAspect.checked).toBe(false)

        els = node.querySelectorAll('.videos-grid-aspect-ratio')
        expect(els.length).toBe(0)

        els = node.querySelectorAll('.videos-grid-flex')
        expect(els.length).toBe(1)

        TestUtils.Simulate.change(radioAspect)

        expect(radioAuto.checked).toBe(false)
        expect(radioLegacy.checked).toBe(false)
        expect(radioAspect.checked).toBe(true)

        els = node.querySelectorAll('.videos-grid-aspect-ratio')
        expect(els.length).toBe(1)

        els = node.querySelectorAll('.videos-grid-flex')
        expect(els.length).toBe(0)
      })
    })

    describe('video menu', () => {
      beforeEach(() => {
        dispatchSpy.mockClear()
      })

      it('minimizes the video on "Maximize" click', () => {
        let maximized = node.querySelectorAll('.videos-grid video')
        expect(maximized.length).toBe(2)

        let minimized = node.querySelectorAll('.videos-toolbar video')
        expect(minimized.length).toBe(0)

        const item = node.querySelector('.dropdown .action-minimize')!
        expect(item).toBeTruthy()
        TestUtils.Simulate.click(item)
        expect(dispatchSpy.mock.calls).toEqual([[{
          type: constants.MINIMIZE_TOGGLE,
          payload: {
            peerId: constants.ME,
            streamId: store.getState()
            .streams.localStreams[StreamTypeCamera]!.streamId,
          },
        }]])

        maximized = node.querySelectorAll('.videos-grid video')
        expect(maximized.length).toBe(1)

        minimized = node.querySelectorAll('.videos-toolbar video')
        expect(minimized.length).toBe(1)

        TestUtils.Simulate.click(node.querySelector('.sidebar-menu-settings')!)

        // Test that the toolbar shows and hides on checkbox click
        let checkbox = node.querySelector(
          '.sidebar .settings .settings-show-minimized-toolbar-toggle')!
        expect(checkbox).toBeTruthy()
        TestUtils.Simulate.change(checkbox)

        // TODO assert class name change instead since we just hide the toolbar
        // now.
        minimized = node.querySelectorAll('.videos-toolbar video')
        expect(minimized.length).toBe(1)

        TestUtils.Simulate.change(checkbox)

        minimized = node.querySelectorAll('.videos-toolbar video')
        expect(minimized.length).toBe(1)

        // Test that the video can be unminimized
        TestUtils.Simulate.click(node.querySelector('.sidebar-menu-users')!)

        checkbox = node.querySelector('.sidebar .users li input')!
        expect(checkbox).toBeTruthy()
        TestUtils.Simulate.change(checkbox)

        maximized = node.querySelectorAll('.videos-grid video')
        expect(maximized.length).toBe(2)

        minimized = node.querySelectorAll('.videos-toolbar video')
        expect(minimized.length).toBe(0)
      })

      it('toggles object-fit on "Toggle Fit" click', () => {
        ['contain', ''].forEach(objectFit => {
          const item = node.querySelector('.dropdown .action-toggle-fit')!
          expect(item).toBeTruthy()
          TestUtils.Simulate.click(item)
          const video = node.querySelector('video')! as HTMLVideoElement
          expect((video).style.objectFit).toBe(objectFit)
          expect(dispatchSpy.mock.calls).toEqual([])
        })
      })
    })

  })

})
