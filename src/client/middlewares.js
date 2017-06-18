import logger from 'redux-logger'
import promiseMiddleware from 'redux-promise-middleware'
import thunk from 'redux-thunk'

export const middlewares = [thunk, promiseMiddleware()]
export const create = log => {
  const m = middlewares.slice()
  log && m.push(logger)
  return m
}
