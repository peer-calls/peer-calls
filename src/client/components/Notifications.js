import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import { CSSTransitionGroup } from 'react-transition-group'

export const NotificationPropTypes = PropTypes.shape({
  id: PropTypes.string.isRequired,
  type: PropTypes.string.isRequired,
  message: PropTypes.string.isRequired
})

export default class Notifications extends React.PureComponent {
  static propTypes = {
    notifications: PropTypes.objectOf(NotificationPropTypes).isRequired,
    max: PropTypes.number.isRequired
  }
  static defaultProps = {
    max: 10
  }
  render () {
    const { notifications, max } = this.props
    return (
      <div className="notifications">
        <CSSTransitionGroup
          transitionEnterTimeout={200}
          transitionLeaveTimeout={100}
          transitionName="fade"
        >
          {Object.keys(notifications).slice(-max).map(id => (
            <div
              className={classnames(notifications[id].type, 'notification')}
              key={id}
            >
              {notifications[id].message}
            </div>
          ))}
        </CSSTransitionGroup>
      </div>
    )
  }
}
