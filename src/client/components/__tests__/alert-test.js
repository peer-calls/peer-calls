jest.unmock('../alert.js')

const React = require('react')
const ReactDOM = require('react-dom')
const TestUtils = require('react-addons-test-utils')

const Alert = require('../alert.js')
const dispatcher = require('../../dispatcher/dispatcher.js')
const alertStore = require('../../store/alertStore.js')

describe('alert', () => {
  beforeEach(() => {
    alertStore.getAlert.mockClear()
  })

  function render () {
    let component = TestUtils.renderIntoDocument(<div><Alert /></div>)
    return ReactDOM.findDOMNode(component)
  }

  describe('render', () => {
    it('should do nothing when no alert', () => {
      let node = render()
      expect(node.querySelector('.alert.hidden')).toBeTruthy()
    })

    it('should render alert', () => {
      alertStore.getAlert.mockReturnValue({
        message: 'example',
        type: 'warning'
      })

      let node = render()

      expect(node.querySelector('.alert.warning')).toBeTruthy()
      expect(node.querySelector('.alert span').textContent).toMatch(/example/)
      expect(node.querySelector('.alert button')).toBeNull()
    })

    it('should render dismissable alert', () => {
      alertStore.getAlert.mockReturnValue({
        message: 'example',
        type: 'warning',
        dismissable: true
      })

      let node = render()

      expect(node.querySelector('.alert.warning')).toBeTruthy()
      expect(node.querySelector('.alert span').textContent).toMatch(/example/)
      expect(node.querySelector('.alert button')).toBeTruthy()
    })

    it('should dispatch dismiss alert on dismiss clicked', () => {
      let alert = {
        message: 'example',
        type: 'warning',
        dismissable: true
      }
      alertStore.getAlert.mockReturnValue(alert)

      let node = render()
      TestUtils.Simulate.click(node.querySelector('.alert button'))

      expect(dispatcher.dispatch.mock.calls).toEqual([[{
        type: 'alert-dismiss',
        alert
      }]])
    })
  })
})
