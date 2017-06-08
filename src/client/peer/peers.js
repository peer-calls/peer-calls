const _ = require('underscore')
const Peer = require('./Peer.js')
const debug = require('debug')('peer-calls:peer')
const dispatcher = require('../dispatcher/dispatcher.js')
const iceServers = require('../iceServers.js')
const notify = require('../action/notify.js')

let peers = {}

/**
 * @param {Socket} socket
 * @param {User} user
 * @param {String} user.id
 * @param {Boolean} [initiator=false]
 * @param {MediaStream} [stream]
 */
function create ({ socket, user, initiator, stream }) {
  debug('create peer: %s, stream:', user.id, stream)
  notify.warn('Connecting to peer...')

  if (peers[user.id]) {
    notify.info('Cleaning up old connection...')
    destroy(user.id)
  }

  let peer = peers[user.id] = Peer.init({
    initiator: socket.id === initiator,
    stream,
    config: { iceServers }
  })

  peer.once('error', err => {
    debug('peer: %s, error %s', user.id, err.stack)
    notify.error('A peer connection error occurred')
    destroy(user.id)
  })

  peer.on('signal', signal => {
    debug('peer: %s, signal: %o', user.id, signal)

    let payload = { userId: user.id, signal }
    socket.emit('signal', payload)
  })

  peer.once('connect', () => {
    debug('peer: %s, connect', user.id)
    notify.warn('Peer connection established')
    dispatcher.dispatch({ type: 'play' })
  })

  peer.on('stream', stream => {
    debug('peer: %s, stream', user.id)
    dispatcher.dispatch({
      type: 'add-stream',
      userId: user.id,
      stream
    })
  })

  peer.on('data', object => {
    object = JSON.parse(new window.TextDecoder('utf-8').decode(object))
    debug('peer: %s, message: %o', user.id, object)
    notify.info('' + user.id + ': ' + object.message)
  })

  peer.once('close', () => {
    debug('peer: %s, close', user.id)
    notify.error('Peer connection closed')
    dispatcher.dispatch({
      type: 'remove-stream',
      userId: user.id
    })

    // make sure some other peer with different id didn't take place between
    // calling `destroy()` and `close` event
    if (peers[user.id] === peer) delete peers[user.id]
  })
}

function get (userId) {
  return peers[userId]
}

function getIds () {
  return _.map(peers, (peer, id) => id)
}

function clear () {
  debug('clear')
  _.each(peers, (_, userId) => destroy(userId))
  peers = {}
}

function destroy (userId) {
  debug('destroy peer: %s', userId)
  let peer = peers[userId]
  if (!peer) return debug('peer: %s peer not found', userId)
  peer.destroy()
  delete peers[userId]
}

function message (message) {
  message = JSON.stringify({ message })
  _.each(peers, peer => peer.send(message))
}

module.exports = { create, get, getIds, destroy, clear, message }
