import { Action, applyMiddleware, createStore as _createStore, Store as ReduxStore } from 'redux'
import { ThunkAction, ThunkDispatch } from 'redux-thunk'
import { create } from './middlewares'
import reducers from './reducers'

export const middlewares = create(
  window.localStorage && window.localStorage.log,
)

export const createStore = () => _createStore(
  reducers,
  applyMiddleware(...middlewares),
)

export default createStore()

export type Store = ReturnType<typeof createStore>

type TGetState<T> = T extends ReduxStore<infer State> ? State : never
export type State = TGetState<Store>
export type GetState = () => State

export type Dispatch = ThunkDispatch<State, undefined, Action>
export type ThunkResult<R> = ThunkAction<R, State, undefined, Action>
