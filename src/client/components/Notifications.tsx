import CSSTransition from 'react-transition-group/CSSTransition'
import React from 'react'
import classnames from 'classnames'
import { Notification } from '../actions/NotifyActions'

export interface NotificationProps {
  notifications: Record<string, Notification>
  max: number
}

const transitionTimeout = {
  enter: 200,
  exit: 100,
}

export default class Notifications
extends React.PureComponent<NotificationProps> {
  static defaultProps = {
    max: 10,
  }
  render () {
    const { notifications, max } = this.props
    return (
      <div className="notifications">
        <CSSTransition
          classNames='fade'
          timeout={transitionTimeout}
        >
          {Object.keys(notifications).slice(-max).map(id => (
            <div
              className={classnames(notifications[id].type, 'notification')}
              key={id}
            >
              {notifications[id].message}
            </div>
          ))}
        </CSSTransition>
      </div>
    )
  }
}
