import { AsyncAction, PendingAction, ResolvedAction, RejectedAction } from './action'

export function reduce<State, T extends string, P>(
  state: State,
  action: AsyncAction<T, P>,
  handlePending: (state: State, action: PendingAction<T, P>) => State,
  handleResolved: (state: State, action: ResolvedAction<T, P>) => State,
  handleRejected: (state: State, action: RejectedAction<T>) => State,
): State {
  switch (action.status) {
    case 'pending':
      return handlePending(state, action)
    case 'resolved':
      return handleResolved(state, action)
    case 'rejected':
      return handleRejected(state, action)
  }
}
