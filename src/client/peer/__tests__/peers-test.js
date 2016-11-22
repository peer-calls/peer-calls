jest.unmock('../peers.js');

const EventEmitter = require('events').EventEmitter;
const Peer = require('../Peer.js');
const dispatcher = require('../../dispatcher/dispatcher.js');
const notify = require('../../action/notify.js');
const peers = require('../peers.js');

describe('peers', () => {

  function createSocket() {
    let socket = new EventEmitter();
    socket.id = 'user1';
    return socket;
  }

  let socket, stream, peerInstances, user;
  beforeEach(() => {
    dispatcher.dispatch.mockClear();
    notify.warn.mockClear();

    user = { id: 'user2' };
    socket = createSocket();
    peerInstances = [];
    stream = { stream: true };

    Peer.init = jest.genMockFunction().mockImplementation(() => {
      let peer = new EventEmitter();
      peer.destroy = jest.genMockFunction();
      peer.signal = jest.genMockFunction();
      peerInstances.push(peer);
      return peer;
    });
  });

  afterEach(() => peers.clear());

  describe('create', () => {

    it('creates a new peer', () => {
      peers.create({ socket, user, initiator: 'user2', stream });

      expect(notify.warn.mock.calls).toEqual([[ 'Connecting to peer...' ]]);

      expect(peerInstances.length).toBe(1);
      expect(Peer.init.mock.calls.length).toBe(1);
      expect(Peer.init.mock.calls[0][0].initiator).toBe(false);
      expect(Peer.init.mock.calls[0][0].stream).toBe(stream);
    });

    it('sets initiator correctly', () => {
      peers.create({ socket, user, initiator: 'user1', stream });

      expect(peerInstances.length).toBe(1);
      expect(Peer.init.mock.calls.length).toBe(1);
      expect(Peer.init.mock.calls[0][0].initiator).toBe(true);
      expect(Peer.init.mock.calls[0][0].stream).toBe(stream);
    });

    it('destroys old peer before creating new one', () => {
      peers.create({ socket, user, initiator: 'user2', stream });
      peers.create({ socket, user, initiator: 'user2', stream });

      expect(peerInstances.length).toBe(2);
      expect(Peer.init.mock.calls.length).toBe(2);
      expect(peerInstances[0].destroy.mock.calls.length).toBe(1);
      expect(peerInstances[1].destroy.mock.calls.length).toBe(0);
    });

  });

  describe('events', () => {

    let peer;

    beforeEach(() => {
      peers.create({ socket, user, initiator: 'user1', stream });
      notify.warn.mockClear();
      peer = peerInstances[0];
    });

    describe('connect', () => {

      beforeEach(() => peer.emit('connect'));

      it('sends a notification', () => {
        expect(notify.warn.mock.calls).toEqual([[
          'Peer connection established'
        ]]);
      });

      it('dispatches "play" action', () => {
        expect(dispatcher.dispatch.mock.calls).toEqual([[{ type: 'play' }]]);
      });

    });

  });

  describe('get', () => {

    it('returns undefined when not found', () => {
      expect(peers.get(user.id)).not.toBeDefined();
    });

    it('returns Peer instance when found', () => {
      peers.create({ socket, user, initiator: 'user2', stream });

      expect(peers.get(user.id)).toBe(peerInstances[0]);
    });

  });

  describe('getIds', () => {

    it('returns ids of all peers', () => {
      peers.create({
        socket, user: {id: 'user2' }, initiator: 'user2', stream
      });
      peers.create({
        socket, user: {id: 'user3' }, initiator: 'user3', stream
      });

      expect(peers.getIds()).toEqual([ 'user2', 'user3' ]);
    });

  });

  describe('destroy', () => {

    it('destroys a peer and removes it', () => {
      peers.create({ socket, user, initiator: 'user2', stream });

      peers.destroy(user.id);

      expect(peerInstances[0].destroy.mock.calls.length).toEqual(1);
    });

    it('throws no error when peer missing', () => {
      peers.destroy('bla123');
    });

  });

  describe('clear', () => {

    it('destroys all peers and removes them', () => {
      peers.create({
        socket, user: {id: 'user2' }, initiator: 'user2', stream
      });
      peers.create({
        socket, user: {id: 'user3' }, initiator: 'user3', stream
      });

      peers.clear();

      expect(peerInstances[0].destroy.mock.calls.length).toEqual(1);
      expect(peerInstances[1].destroy.mock.calls.length).toEqual(1);

      expect(peers.getIds()).toEqual([]);
    });

  });

});
