jest.unmock('../notifications.js');

const React = require('react');
const ReactDOM = require('react-dom');
const TestUtils = require('react-addons-test-utils');

const Notifications = require('../notifications.js');
const notificationsStore = require('../../store/notificationsStore.js');

describe('alert', () => {

  beforeEach(() => {
    notificationsStore.getNotifications.mockClear();
    notificationsStore.getNotifications.mockReturnValue([]);
  });

  function render(component) {
    let rendered = TestUtils.renderIntoDocument(<div>{component}</div>);
    return ReactDOM.findDOMNode(rendered);
  }

  describe('render', () => {

    it('should render notifications placeholder', () => {
      let node = render(<Notifications />);
      expect(node.querySelector('.notifications')).toBeTruthy();
      expect(node.querySelector('.notifications .notification')).toBeFalsy();
    });

    it('should render notifications', () => {
      notificationsStore.getNotifications.mockReturnValue([{
        _id: 1,
        message: 'message 1',
        type: 'warning'
      }, {
        _id: 2,
        message: 'message 2',
        type: 'error'
      }]);

      let node = render(<Notifications />);
      expect(notificationsStore.getNotifications.mock.calls).toEqual([[ 10 ]]);

      let c = node.querySelector('.notifications');
      expect(c).toBeTruthy();
      expect(c.querySelectorAll('.notification').length).toBe(2);
      expect(c.querySelector('.notification.warning').textContent)
      .toEqual('message 1');
      expect(c.querySelector('.notification.error').textContent)
      .toEqual('message 2');
    });

    it('should render max X notifications', () => {
      render(<Notifications max={1} />);
      expect(notificationsStore.getNotifications.mock.calls).toEqual([[ 1 ]]);
    });
  });

});
