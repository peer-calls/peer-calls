jest.unmock('../alertStore.js')

const dispatcher = require('../../dispatcher/dispatcher.js')
const alertStore = require('../alertStore.js')

describe('alertStore', () => {
  let handleAction, onChange
  beforeEach(() => {
    handleAction = dispatcher.register.mock.calls[0][0]
    handleAction({ type: 'alert-clear' })

    onChange = jest.genMockFunction()
    alertStore.addListener(onChange)
  })
  afterEach(() => {
    alertStore.removeListener(onChange)
  })

  describe('alert', () => {
    it('should add alerts to end of queue and dispatch change', () => {
      let alert1 = { message: 'example alert 1' }
      let alert2 = { message: 'example alert 2' }

      handleAction({ type: 'alert', alert: alert1 })
      handleAction({ type: 'alert', alert: alert2 })

      expect(onChange.mock.calls.length).toBe(2)
      expect(alertStore.getAlerts()).toEqual([ alert1, alert2 ])
      expect(alertStore.getAlert()).toBe(alert1)
    })
  })

  describe('alert-dismiss', () => {
    it('should remove alert and dispatch change', () => {
      let alert1 = { message: 'example alert 1' }
      let alert2 = { message: 'example alert 2' }

      handleAction({ type: 'alert', alert: alert1 })
      handleAction({ type: 'alert', alert: alert2 })
      handleAction({ type: 'alert-dismiss', alert: alert1 })

      expect(onChange.mock.calls.length).toBe(3)
      expect(alertStore.getAlerts()).toEqual([ alert2 ])
      expect(alertStore.getAlert()).toBe(alert2)
    })
  })

  describe('alert-clear', () => {
    it('should remove all alerts', () => {
      let alert1 = { message: 'example alert 1' }
      let alert2 = { message: 'example alert 2' }

      handleAction({ type: 'alert', alert: alert1 })
      handleAction({ type: 'alert', alert: alert2 })
      handleAction({ type: 'alert-clear' })

      expect(onChange.mock.calls.length).toBe(3)
      expect(alertStore.getAlerts()).toEqual([])
      expect(alertStore.getAlert()).not.toBeDefined()
    })
  })
})
