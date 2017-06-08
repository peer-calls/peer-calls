jest.unmock('../notify.js')

const dispatcher = require('../../dispatcher/dispatcher.js')
const notify = require('../notify.js')

describe('notify', () => {
  beforeEach(() => dispatcher.dispatch.mockClear())

  describe('info', () => {
    it('should dispatch info notification', () => {
      notify.info('test: {0} {1}', 'arg1', 'arg2')

      expect(dispatcher.dispatch.mock.calls).toEqual([[{
        type: 'notify',
        notification: {
          message: 'test: arg1 arg2',
          type: 'info'
        }
      }]])
    })
  })

  describe('warn', () => {
    it('should dispatch warning notification', () => {
      notify.warn('test: {0} {1}', 'arg1', 'arg2')

      expect(dispatcher.dispatch.mock.calls).toEqual([[{
        type: 'notify',
        notification: {
          message: 'test: arg1 arg2',
          type: 'warning'
        }
      }]])
    })
  })

  describe('error', () => {
    it('should dispatch error notification', () => {
      notify.error('test: {0} {1}', 'arg1', 'arg2')

      expect(dispatcher.dispatch.mock.calls).toEqual([[{
        type: 'notify',
        notification: {
          message: 'test: arg1 arg2',
          type: 'error'
        }
      }]])
    })
  })

  describe('alert', () => {
    it('should dispatch an alert', () => {
      notify.alert('alert!', true)

      expect(dispatcher.dispatch.mock.calls).toEqual([[{
        type: 'alert',
        alert: {
          action: 'Dismiss',
          dismissable: true,
          message: 'alert!',
          type: 'warning'
        }
      }]])
    })
  })
})
