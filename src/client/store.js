import { create } from './middlewares.js'
import reducers from './reducers'
import { applyMiddleware, createStore } from 'redux'
export const middlewares = create(
  window.localStorage && window.localStorage.log
)

export default createStore(
  reducers,
  applyMiddleware.apply(null, middlewares)
)
