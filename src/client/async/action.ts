import { Action } from 'redux'

export type PendingAction<T extends string, P> = Action<T> & Promise<P> & {
  status: 'pending'
}

export type ResolvedAction<T extends string, P> = Action<T> & {
  payload: P
  status: 'resolved'
}

export type RejectedAction<T extends string> = Action<T> & {
  payload: Error
  status: 'rejected'
}

export function isRejectedAction(
  value: unknown,
): value is RejectedAction<string> {
  // eslint-disable-next-line
  const v = value as any
  return !!v && 'type' in v && typeof v.type === 'string' &&
    'status' in v && v.status === 'rejected'
}

export type AsyncAction<T extends string, P> =
  PendingAction<T, P> |
  ResolvedAction<T, P> |
  RejectedAction<T>

export type GetAsyncAction<A> =
  A extends PendingAction<infer T, infer P>
  ? AsyncAction<T, P>
  : A extends ResolvedAction<infer T, infer P>
  ? AsyncAction<T, P>
  : never

export type GetAllActions<T> = {
  [K in keyof T]: T[K] extends (...args: any[]) => infer R
  ? R
  : never
}[keyof T]

export type GetAllAsyncActions<T> = GetAsyncAction<GetAllActions<T>>

function isPromise(value: unknown): value is Promise<unknown> {
  return value && typeof value === 'object' &&
    typeof (value as Promise<unknown>).then === 'function'
}

export function isPendingAction(
  value: unknown,
): value is PendingAction<string, unknown> {
  return isPromise(value) &&
    typeof (value as unknown as { type: 'string' }).type === 'string'
}

export function makeAction<A extends unknown[], T extends string, P>(
  type: T,
  impl: (...args: A) => Promise<P>,
): (...args: A) => PendingAction<T, P> {
  return (...args: A) => {
    const pendingAction = impl(...args) as PendingAction<T, P>
    pendingAction.type = type
    pendingAction.status = 'pending'
    return pendingAction
  }
}
