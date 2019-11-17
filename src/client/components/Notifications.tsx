import CSSTransition from 'react-transition-group/CSSTransition'
import TransitionGroup from 'react-transition-group/TransitionGroup'
import React from 'react'
import classnames from 'classnames'
import { dismissNotification, Notification } from '../actions/NotifyActions'

export interface NotificationsProps {
  notifications: Record<string, Notification>
  dismiss: typeof dismissNotification
  max: number
}

const transitionTimeout = {
  enter: 200,
  exit: 100,
}

export interface NotificationProps {
  notification: Notification
  dismiss: typeof dismissNotification
  timeout: number
}

const Notification = React.memo(
  function Notification(props: NotificationProps) {
    const { dismiss, notification } = props
    React.useEffect(() => {
      const timeout = setTimeout(dismiss, props.timeout, notification.id)
      return () => {
        clearTimeout(timeout)
        dismiss(notification.id)
      }
    }, [])
    return (
      <div className={classnames(notification.type, 'notification')}>
        {notification.message}
      </div>
    )
  },
)

export default class Notifications
extends React.PureComponent<NotificationsProps> {
  static defaultProps = {
    max: 5,
  }
  render () {
    const { dismiss, notifications, max } = this.props
    return (
      <div className="notifications">
        <TransitionGroup>
          {Object.keys(notifications).slice(-max).map(id => (
            <CSSTransition
              key={id}
              classNames='fade'
              timeout={transitionTimeout}
            >
                <Notification
                  notification={notifications[id]}
                  dismiss={dismiss}
                  timeout={10000}
                />
            </CSSTransition>
          ))}
        </TransitionGroup>
      </div>
    )
  }
}
