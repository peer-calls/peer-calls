'use strict'

const EventEmitter = require('events').EventEmitter
const handleSocket = require('../socket.js')

describe('server/socket', () => {
  let socket, io, rooms
  beforeEach(() => {
    socket = new EventEmitter()
    socket.id = 'socket0'
    socket.join = jest.fn()
    socket.leave = jest.fn()
    rooms = {}

    io = {}
    io.in = io.to = jest.fn().mockImplementation(room => {
      return (rooms[room] = rooms[room] || {
        emit: jest.fn()
      })
    })

    io.sockets = {
      adapter: {
        rooms: {
          room1: {
            'socket0': true
          },
          room2: {
            'socket0': true
          },
          room3: {
            sockets: {
              'socket0': true,
              'socket1': true,
              'socket2': true
            }
          }
        }
      }
    }

    socket.leave = jest.fn()
    socket.join = jest.fn()
  })

  it('should be a function', () => {
    expect(typeof handleSocket).toBe('function')
  })

  describe('socket events', () => {
    beforeEach(() => handleSocket(socket, io))

    describe('signal', () => {
      it('should broadcast signal to specific user', () => {
        let signal = { type: 'signal' }

        socket.emit('signal', { userId: 'a', signal })

        expect(io.to.mock.calls).toEqual([[ 'a' ]])
        expect(io.to('a').emit.mock.calls).toEqual([[
          'signal', {
            userId: 'socket0',
            signal
          }
        ]])
      })
    })

    describe('ready', () => {
      it('should call socket.leave if socket.room', () => {
        socket.room = 'room1'
        socket.emit('ready', 'room2')

        expect(socket.leave.mock.calls).toEqual([[ 'room1' ]])
        expect(socket.join.mock.calls).toEqual([[ 'room2' ]])
      })

      it('should call socket.join to room', () => {
        socket.emit('ready', 'room3')
        expect(socket.join.mock.calls).toEqual([[ 'room3' ]])
      })

      it('should emit users', () => {
        socket.emit('ready', 'room3')

        expect(io.to.mock.calls).toEqual([[ 'room3' ]])
        expect(io.to('room3').emit.mock.calls).toEqual([[
          'users', {
            initiator: 'socket0',
            users: [{
              id: 'socket0'
            }, {
              id: 'socket1'
            }, {
              id: 'socket2'
            }]
          }
        ]])
      })
    })
  })
})
