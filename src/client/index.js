'use strict'
import App from './containers/App.js'
import React from 'react'
import ReactDOM from 'react-dom'
import store from './store.js'
import { Provider } from 'react-redux'
import { play } from './window.js'

const component = (
  <Provider store={store}>
    <App />
  </Provider>
)

ReactDOM.render(component, document.getElementById('container'))
play()
