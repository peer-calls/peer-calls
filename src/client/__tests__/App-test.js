jest.mock('../actions/CallActions.js')
jest.mock('../callId.js')
jest.mock('../iceServers.js')

import App from '../containers/App.js'
import React from 'react'
import ReactDOM from 'react-dom'
import TestUtils from 'react-dom/test-utils'
import configureStore from 'redux-mock-store'
import reducers from '../reducers'
import { Provider } from 'react-redux'
import { init } from '../actions/CallActions.js'
import { middlewares } from '../store.js'

describe('App', () => {

  let state
  beforeEach(() => {
    init.mockReturnValue({ type: 'INIT' })
    state = reducers()
  })

  let component, node, store
  function render() {
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

})
