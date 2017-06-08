const EventEmitter = require('events')
const debug = require('debug')('peer-calls:alertStore')
const dispatcher = require('../dispatcher/dispatcher.js')

const emitter = new EventEmitter()
const addListener = cb => emitter.on('change', cb)
const removeListener = cb => emitter.removeListener('change', cb)

let alerts = []

const handlers = {
  alert: ({ alert }) => {
    debug('alert: %s', alert.message)
    alerts.push(alert)
  },
  'alert-dismiss': ({ alert }) => {
    debug('alert-dismiss: %s', alert.message)
    let index = alerts.indexOf(alert)
    debug('index: %s', index)
    if (index < 0) return
    alerts.splice(index, 1)
  },
  'alert-clear': () => {
    debug('alert-clear')
    alerts = []
  }
}

const dispatcherIndex = dispatcher.register(action => {
  let handle = handlers[action.type]
  if (!handle) return
  handle(action)
  emitter.emit('change')
})

function getAlert () {
  return alerts[0]
}

function getAlerts () {
  return alerts
}

module.exports = {
  dispatcherIndex,
  addListener,
  removeListener,
  getAlert,
  getAlerts
}
