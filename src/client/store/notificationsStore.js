const EventEmitter = require('events')
const debug = require('debug')('peer-calls:alertStore')
const dispatcher = require('../dispatcher/dispatcher.js')

const emitter = new EventEmitter()
const addListener = cb => emitter.on('change', cb)
const removeListener = cb => emitter.removeListener('change', cb)

let index = 0
let notifications = []

function dismiss (notification) {
  let index = notifications.indexOf(notification)
  if (index < 0) return
  notifications.splice(index, 1)
  clearTimeout(notification._timeout)
  delete notification._timeout
}

function emitChange () {
  emitter.emit('change')
}

const handlers = {
  notify: ({ notification }) => {
    index++
    debug('notify', notification.message)
    notification._id = index
    notifications.push(notification)
    notification._timeout = setTimeout(() => {
      debug('notify-dismiss timeout: %s', notification.message)
      dismiss(notification)
      emitChange()
    }, 10000)
  },
  'notify-dismiss': ({ notification }) => {
    debug('notify-dismiss: %s', notification.message)
    dismiss(notification)
  },
  'notify-clear': () => {
    debug('notify-clear')
    notifications = []
  }
}

const dispatcherIndex = dispatcher.register(action => {
  let handle = handlers[action.type]
  if (!handle) return
  handle(action)
  emitChange()
})

function getNotifications (max) {
  if (!max) max = notifications.length
  let start = notifications.length - max
  if (start < 0) start = 0
  return notifications.slice(start, notifications.length)
}

module.exports = {
  dispatcherIndex,
  addListener,
  removeListener,
  getNotifications
}
