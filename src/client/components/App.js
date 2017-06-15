import Alerts, { AlertPropType } from './Alerts.js'
import Input from './Input.js'
import Notifications from './Notifications.js'
import PropTypes from 'prop-types'
import React from 'react'
import Video, { StreamPropType } from './Video.js'
import _ from 'underscore'

export default class App extends React.Component {
  static propTypes = {
    streams: PropTypes.objectOf(StreamPropType).isRequired,
    alerts: PropTypes.arrayOf(AlertPropType).isRequired,
    activate: PropTypes.func.isRequired,
    active: PropTypes.string,
    init: PropTypes.func.isRequired,
    notify: PropTypes.func.isRequired
  }
  componentDidMount () {
    const { init } = this.props
    init()
  }
  render () {
    const {
      active, activate, alerts, dismiss, notify, notifications, streams
    } = this.props

    return (<div className="app">
      <Alerts alerts={alerts} dismiss={dismiss} />
      <Notifications notifications={notifications} />
      <Input notify={notify} />
      <div className="videos">
        {_.map(streams, (stream, userId) => (
          <Video
            activate={activate}
            active={userId === active}
            stream={stream}
          />
        ))}
      </div>
    </div>)
  }
}
