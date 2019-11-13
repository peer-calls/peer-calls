import '@babel/polyfill'
import App from './containers/App'
import React from 'react'
import ReactDOM from 'react-dom'
import store from './store'
import { Provider } from 'react-redux'
import { play } from './window'

const component = (
  <Provider store={store}>
    <App />
  </Provider>
)

ReactDOM.render(component, document.getElementById('container'))
play()
