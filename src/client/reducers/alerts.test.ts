import * as NotifyActions from '../actions/NotifyActions'
import { applyMiddleware, createStore, Store, bindActionCreators } from 'redux'
import { create } from '../middlewares'
import reducers from './index'
import values from 'lodash/values'

jest.useFakeTimers()

describe('reducers/alerts', () => {

  let store: Store
  let notifyActions: typeof NotifyActions
  beforeEach(() => {
    store = createStore(
      reducers,
      applyMiddleware(...create()),
    )
   notifyActions = bindActionCreators(NotifyActions, store.dispatch)
  })


  describe('clearAlert', () => {

    [true, false].forEach(dismissable => {
      beforeEach(() => {
        notifyActions.clearAlerts()
      })
      it('adds alert to store', () => {
        notifyActions.alert('test', dismissable)
        expect(store.getState().alerts).toEqual([{
          action: dismissable ? 'Dismiss' : undefined,
          dismissable,
          message: 'test',
          type: 'warning',
        }])
      })
    })

  })

  describe('dismissAlert', () => {

    it('removes an alert', () => {
      store.dispatch(NotifyActions.alert('test', true))
      expect(store.getState().alerts.length).toBe(1)
      store.dispatch(NotifyActions.dismissAlert(store.getState().alerts[0]))
      expect(store.getState().alerts.length).toBe(0)
    })

    it('does not remove an alert when not found', () => {
      store.dispatch(NotifyActions.alert('test', true))
      expect(store.getState().alerts.length).toBe(1)
      store.dispatch(NotifyActions.dismissAlert({
        action: undefined,
        dismissable: false,
        message: 'bla',
        type: 'error',
      }))
      expect(store.getState().alerts.length).toBe(1)
    })

  })

  const methods: Array<'info' | 'warning' | 'error'> = [
    'info',
    'warning',
    'error',
  ]

  methods.forEach(type => {

    describe(type, () => {

      beforeEach(() => {
        notifyActions[type]('Hi {0}!', 'John')
      })

      it('adds a notification', () => {
        expect(values(store.getState().notifications)).toEqual([{
          id: jasmine.any(String),
          message: 'Hi John!',
          type,
        }])
      })

      it('dismisses notification after a timeout', () => {
        jest.runAllTimers()
        expect(store.getState().notifications).toEqual({})
      })

      it('does not fail when no arguments', () => {
        notifyActions[type]()
      })

    })

  })

  describe('clear', () => {

    it('clears all alerts', () => {
      notifyActions.info('Hi {0}!', 'John')
      notifyActions.warning('Hi {0}!', 'John')
      notifyActions.error('Hi {0}!', 'John')
      store.dispatch(NotifyActions.clear())
      expect(store.getState().notifications).toEqual({})
    })

  })

})
