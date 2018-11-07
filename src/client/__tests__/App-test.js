jest.mock('../actions/CallActions.js')
jest.mock('../socket.js')
jest.mock('../window.js')

import * as constants from '../constants.js'
import App from '../containers/App.js'
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import configureStore from 'redux-mock-store'
import reducers from '../reducers'
import { MediaStream } from '../window.js'
import { Provider } from 'react-redux'
import { init } from '../actions/CallActions.js'
import { middlewares } from '../store.js'

describe('App', () => {

  const initAction = { type: 'INIT' }

  let state
  beforeEach(() => {
    init.mockReturnValue(initAction)
    state = reducers()
    window.HTMLMediaElement.prototype.play = jest.fn()
  })

  let component, node, store
  function render () {
    store = configureStore(middlewares)(state)
    component = TestUtils.renderIntoDocument(
      <Provider store={store}>
        <App />
      </Provider>
    )
    node = ReactDOM.findDOMNode(component)
  }

  describe('render', () => {
    it('renders without issues', () => {
      render()
      expect(node).toBeTruthy()
      expect(init.mock.calls.length).toBe(1)
    })
  })

  describe('state', () => {
    let alert
    beforeEach(() => {
      state.streams = {
        test: {
          mediaStream: new MediaStream(),
          url: 'blob://'
        }
      }
      state.peers = {
        test: {}
      }
      state.notifications = state.notifications.merge({
        'notification1': {
          id: 'notification1',
          message: 'test',
          type: 'warning'
        }
      })
      const alerts = state.alerts.asMutable()
      alert = {
        dismissable: true,
        action: 'Dismiss',
        message: 'test alert'
      }
      alerts.push(alert)
      state.alerts = alerts
      render()
      store.clearActions()
    })

    describe('alerts', () => {
      it('can be dismissed', () => {
        const dismiss = node.querySelector('.action-alert-dismiss')
        TestUtils.Simulate.click(dismiss)
        expect(store.getActions()).toEqual([{
          type: constants.ALERT_DISMISS,
          payload: alert
        }])
      })
    })

    describe('video', () => {
      it('can be activated', () => {
        const video = node.querySelector('video')
        TestUtils.Simulate.click(video)
        expect(store.getActions()).toEqual([{
          type: constants.ACTIVE_TOGGLE,
          payload: { userId: constants.ME }
        }])
      })
    })

  })

})
