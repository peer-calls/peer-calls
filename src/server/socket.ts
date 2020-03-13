'use strict'
import _debug from 'debug'
import map from 'lodash/map'
import { ServerSocket, TypedIO } from '../shared'
import { Store } from './store'

const debug = _debug('peercalls:socket')

export default function handleSocket(
  socket: ServerSocket,
  io: TypedIO,
  store: Store,
) {
  socket.once('disconnect', () => {
    if (socket.userId) {
      store.remove(socket.userId)
    }
  })

  socket.on('signal', payload => {
    // debug('signal: %s, payload: %o', socket.userId, payload)
    const socketId = store.get(payload.userId)
    if (socketId) {
      io.to(socketId).emit('signal', {
        userId: socket.userId,
        signal: payload.signal,
      })
    }
  })

  socket.on('ready', payload => {
    const { userId, room } = payload
    debug('ready: %s, room: %s', userId, room)
    if (socket.room) socket.leave(socket.room)
    socket.userId = userId
    store.set(userId, socket.id)
    socket.room = room
    socket.join(room)
    socket.room = room

    const users = getUsers(room)

    debug('ready: %s, room: %s, users: %o', userId, room, users)

    io.to(room).emit('users', {
      initiator: userId,
      users,
    })
  })

  function getUsers (room: string) {
    return map(io.sockets.adapter.rooms[room].sockets, (_, socketId) => {
      const userSocket = io.sockets.sockets[socketId] as ServerSocket
      return {
        socketId: socketId,
        userId: userSocket.userId,
      }
    })
  }

}
