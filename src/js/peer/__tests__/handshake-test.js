jest.dontMock('../handshake.js');
jest.dontMock('events');
jest.dontMock('debug');
jest.dontMock('underscore');

const dispatcher = require('../../dispatcher/dispatcher.js');
const handshake = require('../handshake.js');
const Peer = require('../Peer.js');
const EventEmitter = require('events').EventEmitter;

describe('handshake', () => {

  let socket, peers;
  beforeEach(() => {
    socket = new EventEmitter();
    socket.id = 'a';
    peers = [];

    Peer.init = jest.genMockFunction().mockImplementation(() => {
      let peer = new EventEmitter();
      peer.destroy = jest.genMockFunction();
      peer.signal = jest.genMockFunction();
      peers.push(peer);
      return peer;
    });

    dispatcher.dispatch.mockClear();
  });

  describe('socket events', () => {

    describe('users', () => {

      it('add a peer for each new user and destroy peers for missing', () => {
        handshake.init(socket, 'bla');

        // given
        let payload = {
          users: [{ id: 'a'}, { id: 'b' }],
          initiator: '/#a',
        };
        socket.emit('users', payload);
        expect(peers.length).toBe(2);

        // when
        payload = {
          users: [{ id: 'a'}, { id: 'c' }],
          initiator: '/#c',
        };
        socket.emit('users', payload);

        // then
        expect(peers.length).toBe(3);
        expect(peers[0].destroy.mock.calls.length).toBe(0);
        expect(peers[1].destroy.mock.calls.length).toBe(1);
        expect(peers[2].destroy.mock.calls.length).toBe(0);
      });

    });

    describe('signal', () => {
      let data;
      beforeEach(() => {
        data = {};
        handshake.init(socket, 'bla');
        socket.emit('users', {
          initiator: '#/a',
          users: [{ id: 'a' }]
        });
      });

      it('should forward signal to peer', () => {
        socket.emit('signal', {
          userId: 'a',
          data
        });

        expect(peers.length).toBe(1);
        expect(peers[0].signal.mock.calls.length).toBe(1);
      });

      it('does nothing if no peer', () => {
        socket.emit('signal', {
          userId: 'b',
          data
        });

        expect(peers.length).toBe(1);
        expect(peers[0].signal.mock.calls.length).toBe(0);
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
        users: [{ id: 'a' }],
        initiator: '/#a'
      });
      expect(peers.length).toBe(1);
      peer = peers[0];

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
          expect(payload.userId).toEqual('a');
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
          userId: 'a',
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
          userId: 'a'
        }]]);
      });

    });

  });

});
