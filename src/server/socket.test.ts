import { EventEmitter } from 'events'
import { Socket } from 'socket.io'
import { TypedIO } from '../shared'
import handleSocket from './socket'
import { MemoryStore, Store } from './store'

describe('server/socket', () => {
  type NamespaceMock = Socket & {
    id: string
    room?: string
    join: jest.Mock
    leave: jest.Mock
    emit: jest.Mock
    clients: (callback: (
      err: Error | undefined, clients: string[]
    ) => void) => void
  }

  let socket: NamespaceMock
  let io: TypedIO  & {
    in: jest.Mock<(room: string) => NamespaceMock>
    to: jest.Mock<(room: string) => NamespaceMock>
  }
  let rooms: Record<string, {emit: any}>
  const socket0 = {
    id: 'socket0',
  }
  const socket1 = {
    id: 'socket1',
  }
  const socket2 = {
    id: 'socket2',
  }
  let emitPromise: Promise<void>
  beforeEach(() => {
    socket = new EventEmitter() as NamespaceMock
    socket.id = 'socket0'
    socket.join = jest.fn()
    socket.leave = jest.fn()
    rooms = {}

    let emitResolve: () => void
    emitPromise = new Promise(resolve => {
      emitResolve = resolve
    })

    const socketsByRoom: Record<string, string[]> = {
      room1: [socket0.id],
      room2: [socket0.id],
      room3: [socket0.id, socket1.id, socket2.id],
    }

    io = {} as any
    io.in = io.to = jest.fn().mockImplementation(room => {
      return (rooms[room] = rooms[room] || {
        emit: jest.fn().mockImplementation(() => emitResolve()),
        clients: callback => {
          callback(undefined, socketsByRoom[room] || [])
        },
      } as NamespaceMock)
    })
  })

  it('should be a function', () => {
    expect(typeof handleSocket).toBe('function')
  })

  describe('socket events', () => {
    let stores: {
      userIdBySocketId: Store
      socketIdByUserId: Store
    }
    beforeEach(() => {
      stores = {
        userIdBySocketId: new MemoryStore(),
        socketIdByUserId: new MemoryStore(),
      }
      stores.socketIdByUserId.set('a', socket0.id)
      stores.userIdBySocketId.set(socket0.id, 'a')
      stores.socketIdByUserId.set('b', socket1.id)
      stores.userIdBySocketId.set(socket1.id, 'b')
      stores.socketIdByUserId.set('c', socket2.id)
      stores.userIdBySocketId.set(socket2.id, 'c')
      handleSocket(socket, io, stores)
    })

    describe('signal', () => {
      it('should broadcast signal to specific user', async () => {
        const signal = { type: 'signal' }

        socket.emit('signal', { userId: 'b', signal })
        await emitPromise

        expect(io.to.mock.calls).toEqual([[ socket1.id ]])
        expect((io.to(socket1.id).emit as jest.Mock).mock.calls).toEqual([[
          'signal', {
            userId: 'a',
            signal,
          },
        ]])
      })
    })

    describe('ready', () => {
      it('never calls socket.leave', async () => {
        socket.room = 'room1'
        socket.emit('ready', {
          userId: 'a',
          room: 'room2',
        })
        await emitPromise

        expect(socket.leave.mock.calls).toEqual([])
        expect(socket.join.mock.calls).toEqual([[ 'room2' ]])
      })

      it('should call socket.join to room', async () => {
        socket.emit('ready', {
          userId: 'b',
          room: 'room3',
        })
        await emitPromise
        expect(socket.join.mock.calls).toEqual([[ 'room3' ]])
      })

      it('should emit users', async () => {
        socket.emit('ready', {
          userId: 'a',
          room: 'room3',
        })
        await emitPromise

        // expect(io.to.mock.calls).toEqual([[ 'room3' ]])
        expect((io.to('room3').emit as jest.Mock).mock.calls).toEqual([
          [
            'users', {
              initiator: 'a',
              users: [{
                socketId: socket0.id,
                userId: 'a',
              }, {
                socketId: socket1.id,
                userId: 'b',
              }, {
                socketId: socket2.id,
                userId: 'c',
              }],
            },
          ],
        ])
      })
    })
  })
})
