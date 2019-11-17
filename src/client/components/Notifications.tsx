import CSSTransition from 'react-transition-group/CSSTransition'
import TransitionGroup from 'react-transition-group/TransitionGroup'
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
        <TransitionGroup>
          {Object.keys(notifications).slice(-max).map(id => (
            <CSSTransition
              key={id}
              classNames='fade'
              timeout={transitionTimeout}
            >
              <div
                className={classnames(notifications[id].type, 'notification')}
              >
                {notifications[id].message}
              </div>
            </CSSTransition>
          ))}
        </TransitionGroup>
      </div>
    )
  }
}
