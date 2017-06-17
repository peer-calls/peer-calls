import { create } from './middlewares.js'
import reducers from './reducers'
import { applyMiddleware, createStore } from 'redux'
export const middlewares = create(
  window.localStorage && window.localStorage.debug
)

export default createStore(
  reducers,
  applyMiddleware.apply(null, middlewares)
)
