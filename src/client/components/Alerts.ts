import React from 'react'
import PropTypes from 'prop-types'
import classnames from 'classnames'

export const AlertPropType = PropTypes.shape({
  dismissable: PropTypes.bool,
  action: PropTypes.string.isRequired,
  message: PropTypes.string.isRequired
})

export class Alert extends React.PureComponent {
  static propTypes = {
    alert: AlertPropType,
    dismiss: PropTypes.func.isRequired
  }
  dismiss = () => {
    const { alert, dismiss } = this.props
    dismiss(alert)
  }
  render () {
    const { alert } = this.props

    return (
      <div className={classnames('alert', alert.type)}>
        <span>{alert.message}</span>
        {alert.dismissable && (
          <button
            className="action-alert-dismiss"
            onClick={this.dismiss}
          >{alert.action}</button>
        )}
      </div>
    )
  }
}

export default class Alerts extends React.PureComponent {
  static propTypes = {
    alerts: PropTypes.arrayOf(AlertPropType).isRequired,
    dismiss: PropTypes.func.isRequired
  }
  render () {
    const { alerts, dismiss } = this.props
    return (
      <div className="alerts">
        {alerts.map((alert, i) => (
          <Alert alert={alert} dismiss={dismiss} key={i} />
        ))}
      </div>
    )
  }
}
