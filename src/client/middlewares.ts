import { Middleware } from 'redux'
import logger from 'redux-logger'
import thunk from 'redux-thunk'
import { middleware as asyncMiddleware } from './async'
import { createMessagingMiddleware } from './messaging'

export const middlewares: Middleware[] = [
  thunk,
  asyncMiddleware,
  createMessagingMiddleware(),
]
export const create = (log = false) => {
  const m = middlewares.slice()
  log && m.push(logger)
  return m
}
