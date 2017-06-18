import * as NotifyActions from '../../actions/NotifyActions.js'
import _ from 'underscore'
import { applyMiddleware, createStore } from 'redux'
import { create } from '../../middlewares.js'
import reducers from '../index.js'

jest.useFakeTimers()

describe('reducers/alerts', () => {

  let store
  beforeEach(() => {
    store = createStore(
      reducers,
      applyMiddleware.apply(null, create())
    )
  })

  describe('clearAlert', () => {

    const actions = {
      true: 'Dismiss',
      false: ''
    }
    ;[true, false].forEach(dismissable => {
      beforeEach(() => {
        store.dispatch(NotifyActions.clearAlerts())
      })
      it('adds alert to store', () => {
        store.dispatch(NotifyActions.alert('test', dismissable))
        expect(store.getState().alerts).toEqual([{
          action: actions[dismissable],
          dismissable,
          message: 'test',
          type: 'warning'
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
      store.dispatch(NotifyActions.dismissAlert({}))
      expect(store.getState().alerts.length).toBe(1)
    })

  })

  ;['info', 'warning', 'error'].forEach(type => {

    describe(type, () => {

      beforeEach(() => {
        store.dispatch(NotifyActions[type]('Hi {0}!', 'John'))
      })

      it('adds a notification', () => {
        expect(_.values(store.getState().notifications)).toEqual([{
          id: jasmine.any(String),
          message: 'Hi John!',
          type
        }])
      })

      it('dismisses notification after a timeout', () => {
        jest.runAllTimers()
        expect(store.getState().notifications).toEqual({})
      })

      it('does not fail when no arguments', () => {
        store.dispatch(NotifyActions[type]())
      })

    })

  })

  describe('clear', () => {

    it('clears all alerts', () => {
      store.dispatch(NotifyActions.info('Hi {0}!', 'John'))
      store.dispatch(NotifyActions.warning('Hi {0}!', 'John'))
      store.dispatch(NotifyActions.error('Hi {0}!', 'John'))
      store.dispatch(NotifyActions.clear())
      expect(store.getState().notifications).toEqual({})
    })

  })

})
