import Alerts, { AlertPropType } from './Alerts.js'
import Input from './Input.js'
import Notifications, { NotificationPropTypes } from './Notifications.js'
import PropTypes from 'prop-types'
import React from 'react'
import Video from './Video.js'
import _ from 'underscore'

export default class App extends React.Component {
  static propTypes = {
    dismissAlert: PropTypes.func.isRequired,
    streams: PropTypes.objectOf(PropTypes.string).isRequired,
    alerts: PropTypes.arrayOf(AlertPropType).isRequired,
    setActive: PropTypes.func.isRequired,
    active: PropTypes.string,
    init: PropTypes.func.isRequired,
    notify: PropTypes.func.isRequired,
    notifications: PropTypes.objectOf(NotificationPropTypes).isRequired,
    sendMessage: PropTypes.func.isRequired
  }
  componentDidMount () {
    const { init } = this.props
    init()
  }
  render () {
    const {
      active,
      alerts,
      dismissAlert,
      notifications,
      notify,
      sendMessage,
      setActive,
      streams
    } = this.props

    return (<div className="app">
      <Alerts alerts={alerts} dismiss={dismissAlert} />
      <Notifications notifications={notifications} />
      <Input notify={notify} sendMessage={sendMessage} />
      <div className="videos">
        {_.map(streams, (stream, userId) => (
          <Video
            setActive={setActive}
            active={userId === active}
            key={userId}
            userId={userId}
            stream={stream}
          />
        ))}
      </div>
    </div>)
  }
}
