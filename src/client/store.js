import { create } from './middlewares.js'
import reducers from './reducers'
import { applyMiddleware, createStore as _createStore } from 'redux'
export const middlewares = create(
  window.localStorage && window.localStorage.log
)

export const createStore = () => _createStore(
  reducers,
  applyMiddleware.apply(null, middlewares)
)

export default createStore()
