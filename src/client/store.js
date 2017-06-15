import logger from 'redux-logger'
import promiseMiddleware from 'redux-promise-middleware'
import reducers from './reducers'
import thunk from 'redux-thunk'
import { applyMiddleware, createStore } from 'redux'

export const middlewares = [thunk, promiseMiddleware(), logger]

export default createStore(
  reducers,
  applyMiddleware.apply(null, middlewares)
)
