import { AnyAction, Middleware } from 'redux'
import { isPendingAction, ResolvedAction, RejectedAction } from './action'
import _debug from 'debug'

const debug = _debug('peercalls:async')

export const middleware: Middleware = store => next => (action: AnyAction) => {
  if (!isPendingAction(action)) {
    debug('NOT pending %o', action)
    return next(action)
  }

  debug('Pending: %s %s', action.type, action.status)

  const promise = action
  .then(payload => {
    debug('Resolved: %s resolved', action.type)
    const resolvedAction: ResolvedAction<string, unknown> = {
      payload,
      type: action.type,
      status: 'resolved',
    }
    store.dispatch(resolvedAction)
  })

  // Propagate this action. Only attach listeners to the promise.
  debug('Calling next for %s %s', action.type, action.status)
  next({
    type: action.type,
    status: 'pending',
  })

  const promise2 = promise
  .catch((err: Error) => {
    debug('Rejected: %s rejected %s', action.type, err.stack)
    const rejectedAction: RejectedAction<string> = {
      payload: err,
      type: action.type,
      status: 'rejected',
    }
    store.dispatch(rejectedAction)
  })

  return promise2.then(() => {
    return action
  })
}
