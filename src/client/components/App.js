import Alerts, { AlertPropType } from './Alerts.js'
import * as constants from '../constants.js'
import Input from './Input.js'
import Notifications, { NotificationPropTypes } from './Notifications.js'
import PropTypes from 'prop-types'
import React from 'react'
import Video, { StreamPropType } from './Video.js'
import _ from 'underscore'

export default class App extends React.PureComponent {
  static propTypes = {
    active: PropTypes.string,
    alerts: PropTypes.arrayOf(AlertPropType).isRequired,
    dismissAlert: PropTypes.func.isRequired,
    init: PropTypes.func.isRequired,
    notifications: PropTypes.objectOf(NotificationPropTypes).isRequired,
    notify: PropTypes.func.isRequired,
    peers: PropTypes.object.isRequired,
    sendMessage: PropTypes.func.isRequired,
    streams: PropTypes.objectOf(StreamPropType).isRequired,
    toggleActive: PropTypes.func.isRequired
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
      peers,
      sendMessage,
      toggleActive,
      streams
    } = this.props

    return (<div className="app">
      <Alerts alerts={alerts} dismiss={dismissAlert} />
      <Notifications notifications={notifications} />
      <Input notify={notify} sendMessage={sendMessage} />
      <div className="videos">
        <Video
          active={active === constants.ME}
          onClick={toggleActive}
          stream={streams[constants.ME]}
          userId={constants.ME}
        />

        {_.map(peers, (_, userId) => (
          <Video
            active={userId === active}
            key={userId}
            onClick={toggleActive}
            stream={streams[userId]}
            userId={userId}
          />
        ))}
      </div>
    </div>)
  }
}
