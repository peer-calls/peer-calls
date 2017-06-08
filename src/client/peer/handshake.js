'use strict'
const _ = require('underscore')
const debug = require('debug')('peer-calls:peer')
const notify = require('../action/notify.js')
const peers = require('./peers.js')

function init (socket, roomName, stream) {
  function createPeer (user, initiator) {
    return peers.create({ socket, user, initiator, stream })
  }

  socket.on('signal', payload => {
    let peer = peers.get(payload.userId)
    let signal = payload.signal
    // debug('socket signal, userId: %s, signal: %o', payload.userId, signal);

    if (!peer) return debug('user: %s, no peer found', payload.userId)
    peer.signal(signal)
  })

  socket.on('users', payload => {
    let { initiator, users } = payload
    debug('socket users: %o', users)
    notify.info('Connected users: {0}', users.length)

    users
    .filter(user => !peers.get(user.id) && user.id !== socket.id)
    .forEach(user => createPeer(user, initiator))

    let newUsersMap = _.indexBy(users, 'id')
    peers.getIds()
    .filter(id => !newUsersMap[id])
    .forEach(peers.destroy)
  })

  debug('socket.id: %s', socket.id)
  debug('emit ready for room: %s', roomName)
  notify.info('Ready for connections')
  socket.emit('ready', roomName)
}

module.exports = { init }
