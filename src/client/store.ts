import { create } from './middlewares.js'
import reducers from './reducers'
import { applyMiddleware, createStore as _createStore, Store as ReduxStore } from 'redux'
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
