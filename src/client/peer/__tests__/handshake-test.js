jest.unmock('../handshake.js');
jest.unmock('../peers.js');
jest.unmock('events');
jest.unmock('underscore');

const EventEmitter = require('events').EventEmitter;
const Peer = require('../Peer.js');
const dispatcher = require('../../dispatcher/dispatcher.js');
const handshake = require('../handshake.js');
const peers = require('../peers.js');

describe('handshake', () => {

  let socket, peerInstances;
  beforeEach(() => {
    socket = new EventEmitter();
    socket.id = 'a';
    peerInstances = [];

    Peer.init = jest.genMockFunction().mockImplementation(() => {
      let peer = new EventEmitter();
      peer.destroy = jest.genMockFunction();
      peer.signal = jest.genMockFunction();
      peerInstances.push(peer);
      return peer;
    });

    dispatcher.dispatch.mockClear();
  });

  afterEach(() => peers.clear());

  describe('socket events', () => {

    describe('users', () => {

      it('add a peer for each new user and destroy peers for missing', () => {
        handshake.init(socket, 'bla');

        // given
        let payload = {
          users: [{ id: 'a'}, { id: 'b' }],
          initiator: 'a',
        };
        socket.emit('users', payload);
        expect(peerInstances.length).toBe(1);

        // when
        payload = {
          users: [{ id: 'a'}, { id: 'c' }],
          initiator: 'c',
        };
        socket.emit('users', payload);

        // then
        expect(peerInstances.length).toBe(2);
        expect(peerInstances[0].destroy.mock.calls.length).toBe(1);
        expect(peerInstances[1].destroy.mock.calls.length).toBe(0);
      });

    });

    describe('signal', () => {
      let data;
      beforeEach(() => {
        data = {};
        handshake.init(socket, 'bla');
        socket.emit('users', {
          initiator: 'a',
          users: [{ id: 'a' }, { id: 'b' }]
        });
      });

      it('should forward signal to peer', () => {
        socket.emit('signal', {
          userId: 'b',
          data
        });

        expect(peerInstances.length).toBe(1);
        expect(peerInstances[0].signal.mock.calls.length).toBe(1);
      });

      it('does nothing if no peer', () => {
        socket.emit('signal', {
          userId: 'a',
          data
        });

        expect(peerInstances.length).toBe(1);
        expect(peerInstances[0].signal.mock.calls.length).toBe(0);
      });

    });

  });

  describe('peer events', () => {

    let peer;
    beforeEach(() => {
      let ready = false;
      socket.once('ready', () => { ready = true; });

      handshake.init(socket, 'bla');

      socket.emit('users', {
        initiator: 'a',
        users: [{ id: 'a' }, { id: 'b'}]
      });
      expect(peerInstances.length).toBe(1);
      peer = peerInstances[0];

      expect(ready).toBeDefined();
    });

    describe('error', () => {

      it('destroys peer', () => {
        peer.emit('error', new Error('bla'));
        expect(peer.destroy.mock.calls.length).toBe(1);
      });

    });

    describe('signal', () => {

      it('emits socket signal with user id', done => {
        let signal = { bla: 'bla' };

        socket.once('signal', payload => {
          expect(payload.userId).toEqual('b');
          expect(payload.signal).toBe(signal);
          done();
        });

        peer.emit('signal', signal);
      });

    });

    describe('stream', () => {

      it('adds a stream to streamStore', () => {
        expect(dispatcher.dispatch.mock.calls.length).toBe(0);

        let stream = {};
        peer.emit('stream', stream);

        expect(dispatcher.dispatch.mock.calls.length).toBe(1);
        expect(dispatcher.dispatch.mock.calls).toEqual([[{
          type: 'add-stream',
          userId: 'b',
          stream
        }]]);
      });

    });

    describe('close', () => {

      it('removes stream from streamStore', () => {
        peer.emit('close');

        expect(dispatcher.dispatch.mock.calls.length).toBe(1);
        expect(dispatcher.dispatch.mock.calls).toEqual([[{
          type: 'remove-stream',
          userId: 'b'
        }]]);
      });

    });

  });

});
