import { AnyAction, Middleware } from 'redux'
import { isPendingAction, ResolvedAction, PendingAction, RejectedAction } from './action'

export const middleware: Middleware = store => next => (action: AnyAction) => {
  if (!isPendingAction(action)) {
    return next(action)
  }

  const promise = action
  .then(payload => {
    const resolvedAction: ResolvedAction<string, unknown> = {
      payload,
      type: action.type,
      status: 'resolved',
    }
    store.dispatch(resolvedAction)
  })

  // Propagate this action. Only attach listeners to the promise.
  next({
    type: action.type,
    status: 'pending',
  })

  const promise2 = promise
  .catch((err: Error) => {
    const rejectedAction: RejectedAction<string> = {
      payload: err,
      type: action.type,
      status: 'rejected',
    }
    store.dispatch(rejectedAction)
  })

  return promise2.then(() => action)
}
