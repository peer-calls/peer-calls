jest.unmock('../notificationsStore.js');

const dispatcher = require('../../dispatcher/dispatcher.js');
const store = require('../notificationsStore.js');

describe('store', () => {

  let handleAction, onChange;
  beforeEach(() => {
    dispatcher.dispatch.mockClear();
    handleAction = dispatcher.register.mock.calls[0][0];

    handleAction({ type: 'notify-clear' });

    onChange = jest.genMockFunction();
    store.addListener(onChange);
  });

  describe('notify', () => {

    it('should add notification and dispatch change', () => {
      let notif1 = { message: 'example notif 1' };
      let notif2 = { message: 'example notif 2' };

      handleAction({ type: 'notify', notification: notif1 });
      handleAction({ type: 'notify', notification: notif2 });

      expect(onChange.mock.calls.length).toBe(2);
      expect(store.getNotifications()).toEqual([ notif1, notif2 ]);
      expect(store.getNotifications(1)).toEqual([ notif2 ]);
      expect(store.getNotifications(3)).toEqual([ notif1, notif2 ]);
    });

    it('should add timeout for autoremoval', () => {
      let notif1 = { message: 'example notif 1' };

      handleAction({ type: 'notify', notification: notif1 });

      expect(onChange.mock.calls.length).toBe(1);
      expect(store.getNotifications()).toEqual([ notif1 ]);

      jest.runAllTimers();

      expect(onChange.mock.calls.length).toBe(2);
      expect(store.getNotifications()).toEqual([]);
    });

  });

  describe('notify-dismiss', () => {

    it('should remove notif and dispatch change', () => {
      let notif1 = { message: 'example notif 1' };
      let notif2 = { message: 'example notif 2' };

      handleAction({ type: 'notify', notification: notif1 });
      handleAction({ type: 'notify', notification: notif2 });
      handleAction({ type: 'notify-dismiss', notification: notif1 });

      expect(onChange.mock.calls.length).toBe(3);
      expect(store.getNotifications()).toEqual([ notif2 ]);
      expect(store.getNotifications(2)).toEqual([ notif2 ]);
    });

  });

});
