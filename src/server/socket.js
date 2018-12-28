'use strict'
const debug = require('debug')('peer-calls:socket')
const _ = require('underscore')

const messages = {}

module.exports = function (socket, io) {
  socket.on('signal', payload => {
    // debug('signal: %s, payload: %o', socket.id, payload)
    io.to(payload.userId).emit('signal', {
      userId: socket.id,
      signal: payload.signal
    })
  })

  socket.on('new_message', payload => {
    addMesssage(socket.room, payload)
    io.to(socket.room).emit('new_message', payload)
  })

  socket.on('position', payload => {
    debug('payload: %o', payload)
    io.to(socket.room).emit('position', payload)
  })

  socket.on('ready', roomName => {
    debug('ready: %s, room: %s', socket.id, roomName)
    if (socket.room) socket.leave(socket.room)
    socket.room = roomName
    socket.join(roomName)
    socket.room = roomName

    let users = getUsers(roomName)
    let messages = getMesssages(roomName)

    debug('ready: %s, room: %s, users: %o, messages: %o',
      socket.id, roomName, users, messages)

    io.to(roomName).emit('users', {
      initiator: socket.id,
      users
    })

    io.to(roomName).emit('messages', messages)
  })

  function getUsers (roomName) {
    return _.map(io.sockets.adapter.rooms[roomName].sockets, (_, id) => {
      return { id }
    })
  }

  function getMesssages (roomName) {
    if (_.isUndefined(messages[roomName])) {
      messages[roomName] = []
    }
    return messages[roomName]
  }

  function addMesssage (roomName, payload) {
    getMesssages(roomName).push(payload)
  }
}
