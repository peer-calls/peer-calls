const React = require('react')
const alertStore = require('../store/alertStore.js')
const dispatcher = require('../dispatcher/dispatcher.js')

function alert () {
  let alert = alertStore.getAlert()
  if (!alert) return <div className='alert hidden'><span>&nbsp;</span></div>
  let button

  function dismiss () {
    dispatcher.dispatch({
      type: 'alert-dismiss',
      alert
    })
  }

  if (alert.dismissable) {
    button = <button onClick={dismiss}>{alert.action}</button>
  }

  return (
    <div className={alert.type + ' alert'}>
      <span>{alert.message}</span>
      {button}
    </div>
  )
}

module.exports = alert
