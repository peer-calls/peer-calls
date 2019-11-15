import { Middleware } from 'redux'
import logger from 'redux-logger'
import thunk from 'redux-thunk'
import { middleware as asyncMiddleware } from './async'

export const middlewares: Middleware[] = [thunk, asyncMiddleware]
export const create = (log = false) => {
  const m = middlewares.slice()
  log && m.push(logger)
  return m
}
