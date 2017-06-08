jest.unmock('../app.js')
jest.unmock('underscore')

const React = require('react')
const ReactDOM = require('react-dom')
const TestUtils = require('react-addons-test-utils')

require('../alert.js').mockImplementation(() => <div />)
require('../notifications.js').mockImplementation(() => <div />)
const App = require('../app.js')
const activeStore = require('../../store/activeStore.js')
const dispatcher = require('../../dispatcher/dispatcher.js')
const streamStore = require('../../store/streamStore.js')

describe('app', () => {
  beforeEach(() => {
    dispatcher.dispatch.mockClear()
  })

  function render (active) {
    streamStore.getStreams.mockReturnValue({
      user1: { stream: 1 },
      user2: { stream: 2 }
    })
    let component = TestUtils.renderIntoDocument(<div><App /></div>)
    return ReactDOM.findDOMNode(component).children[0]
  }

  it('should render div.app', () => {
    let node = render()
    expect(node.tagName).toBe('DIV')
    expect(node.className).toBe('app')
  })

  it('should have rendered two videos', () => {
    let node = render()

    expect(node.querySelectorAll('video').length).toBe(2)
  })

  it('should mark .active video', () => {
    activeStore.getActive.mockReturnValue('user1')
    activeStore.isActive.mockImplementation(test => test === 'user1')

    let node = render()
    expect(node.querySelectorAll('.video-container').length).toBe(2)
    expect(node.querySelectorAll('.video-container.active').length).toBe(1)
  })

  it('should dispatch mark-active on video click', () => {
    let node = render()

    TestUtils.Simulate.click(node.querySelectorAll('video')[1])

    expect(dispatcher.dispatch.mock.calls).toEqual([[{
      type: 'mark-active',
      userId: 'user2'
    }]])
  })
})
