jest.unmock('../activeStore.js');

const dispatcher = require('../../dispatcher/dispatcher.js');
const activeStore = require('../activeStore.js');

describe('activeStore', () => {

  let handleAction = dispatcher.register.mock.calls[0][0];
  let onChange = jest.genMockFunction();

  beforeEach(() => {
    onChange.mockClear();
    activeStore.addListener(onChange);
  });
  afterEach(() => activeStore.removeListener(onChange));

  describe('mark-active', () => {

    it('should mark id as active', () => {
      expect(activeStore.getActive()).not.toBeDefined();
      expect(activeStore.isActive('user1')).toBe(false);
      expect(onChange.mock.calls.length).toBe(0);

      handleAction({ type: 'mark-active', userId: 'user1' });

      expect(activeStore.getActive()).toBe('user1');
      expect(activeStore.isActive('user1')).toBe(true);

      expect(onChange.mock.calls.length).toBe(1);
    });

  });

});
