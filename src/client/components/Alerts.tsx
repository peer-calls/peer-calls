import React from 'react'
import classnames from 'classnames'
import { Alert as AlertType } from '../actions/NotifyActions'

export interface AlertProps {
  alert: AlertType
  dismiss: (alert: AlertType) => void
}

export class Alert extends React.PureComponent<AlertProps> {
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

export interface AlertsProps {
  alerts: AlertType[]
  dismiss: (alert: AlertType) => void
}

export default class Alerts extends React.PureComponent<AlertsProps> {
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
