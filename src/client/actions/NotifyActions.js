import * as constants from '../constants.js'

const TIMEOUT = 5000

function format (string, args) {
  string = args
  .reduce((string, arg, i) => string.replace('{' + i + '}', arg), string)
  return string
}

const _notify = (type, args) => dispatch => {
  let string = args[0] || ''
  let message = format(string, Array.prototype.slice.call(args, 1))
  const payload = { type, message }
  dispatch({
    type: constants.NOTIFY,
    payload
  })
  setTimeout(() => {
    dispatch({
      type: constants.NOTIFY_DISMISS,
      payload
    })
  }, TIMEOUT)
}

export const info = function() {
  return dispatch => _notify('info', arguments)(dispatch)
}

export const warn = function() {
  return dispatch => _notify('warning', arguments)(dispatch)
}

export const error = function() {
  return dispatch => _notify('error', arguments)(dispatch)
}

export function alert (message, dismissable) {
  return {
    type: constants.ALERT,
    payload: {
      action: dismissable ? 'Dismiss' : '',
      dismissable: !!dismissable,
      message,
      type: 'warning'
    }
  }
}

export const dismiss = alert => {
  return {
    type: constants.ALERT_DISMISS,
    payload: alert
  }
}
