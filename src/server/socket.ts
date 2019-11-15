'use strict'
import _debug from 'debug'
import map from 'lodash/map'
import { ServerSocket, TypedIO } from '../shared'

const debug = _debug('peercalls:socket')

export default function handleSocket(socket: ServerSocket, io: TypedIO) {
  socket.on('signal', payload => {
    // debug('signal: %s, payload: %o', socket.id, payload)
    io.to(payload.userId).emit('signal', {
      userId: socket.id,
      signal: payload.signal,
    })
  })

  socket.on('ready', roomName => {
    debug('ready: %s, room: %s', socket.id, roomName)
    if (socket.room) socket.leave(socket.room)
    socket.room = roomName
    socket.join(roomName)
    socket.room = roomName

    const users = getUsers(roomName)

    debug('ready: %s, room: %s, users: %o', socket.id, roomName, users)

    io.to(roomName).emit('users', {
      initiator: socket.id,
      users,
    })
  })

  function getUsers (roomName: string) {
    return map(io.sockets.adapter.rooms[roomName].sockets, (_, id) => {
      return { id }
    })
  }

}
