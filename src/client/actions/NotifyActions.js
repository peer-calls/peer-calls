function format (string, args) {
  string = args
  .reduce((string, arg, i) => string.replace('{' + i + '}', arg), string)
  return string
}

function _notify (type, args) {
  let string = args[0] || ''
  let message = format(string, Array.prototype.slice.call(args, 1))
  return {
    type: 'notify',
    payload: { type, message }
  }
}

export function info () {
  return _notify('info', arguments)
}

export function warn () {
  return _notify('warning', arguments)
}

export function error () {
  return _notify('error', arguments)
}

export function alert (message, dismissable) {
  return {
    type: 'alert',
    payload: {
      action: dismissable ? 'Dismiss' : '',
      dismissable: !!dismissable,
      message,
      type: 'warning'
    }
  }
}
