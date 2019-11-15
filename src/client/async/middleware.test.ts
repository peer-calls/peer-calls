import { GetAllAsyncActions, makeAction } from './action'
import { middleware } from './middleware'
import { reduce } from './reducer'
import { createStore, applyMiddleware, combineReducers } from 'redux'

describe('middleware', () => {

  interface State {
    sum: number
    status: 'pending' | 'resolved' | 'rejected'
  }

  const defaultState: State = {
    status: 'resolved',
    sum: 0,
  }

  const actions = {
    add: makeAction('add', async (a: number, b: number) => {
      return {a, b}
    }),
    subtract: makeAction('subtract', async (a: number, b: number) => {
      return {a, b}
    }),
    reject: makeAction('reject', async (a: number, b: number) => {
      throw new Error('Test reject')
    }),
  }

  type Action = GetAllAsyncActions<typeof actions>

  function result(state = defaultState, action: Action): State {
    switch (action.type) {
      case 'add':
        return reduce(
          state,
          action,
          (state, pending) => ({
            status: pending.status,
            sum: state.sum,
          }),
          (state, resolved) => ({
            status: resolved.status,
            sum: resolved.payload.a + resolved.payload.b,
          }),
          (state, rejected) => ({status: rejected.status, sum: 0}),
        )
      case 'subtract':
        return reduce(
          state,
          action,
          (state, pending) => ({
            status: pending.status,
            sum: state.sum,
          }),
          (state, resolved) => ({
            status: resolved.status,
            sum: resolved.payload.a - resolved.payload.b,
          }),
          (state, rejected) => ({status: rejected.status, sum: 0}),
        )
      case 'reject':
        return reduce(
          state,
          action,
          (state, pending) => ({
            status: pending.status,
            sum: state.sum,
          }),
          (state, resolved) => ({
            status: resolved.status,
            sum: 0,
          }),
          (state, rejected) => ({status: rejected.status, sum: 0}),
        )
      default:
        return state
    }
  }

  function getStore() {
    return createStore(
      combineReducers({ result }),
      applyMiddleware(middleware),
    )
  }

  describe('pending and resolved', () => {
    it('makes it easy to dispatch async actions for redux', async () => {
      const store = getStore()
      await store.dispatch(actions.add(1, 2))
      expect(store.getState()).toEqual({
        result: {
          status: 'resolved',
          sum: 3,
        },
      })
      await store.dispatch(actions.subtract(1, 2))
      expect(store.getState()).toEqual({
        result: {
          status: 'resolved',
          sum: -1,
        },
      })
    })
  })

  describe('rejected', () => {
    it('handles rejected actions', async () => {
      const store = getStore()
      let error!: Error
      try {
        await store.dispatch(actions.reject(1, 2))
      } catch (err) {
        error = err
      }
      expect(error).toBeTruthy()
      expect(error.message).toBe('Test reject')
      expect(store.getState()).toEqual({
        result: {
          status: 'rejected',
          sum: 0,
        },
      })
    })
  })

})
