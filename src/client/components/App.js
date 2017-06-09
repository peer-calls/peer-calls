import Alert from './Alerts.js'
import Input from './Input.js'
import Notifications from './Notifications.js'
import PropTypes from 'prop-types'
import React from 'react'
import Video, { StreamPropType } from './Video.js'
import _ from 'underscore'

export default class App extends React.PureComponent {
  static propTypes = {
    streams: PropTypes.arrayOf(StreamPropType).isRequired,
    activate: PropTypes.func.isRequired,
    active: PropTypes.string.isRequired,
    init: PropTypes.func.isRequired
  }
  componentDidMount () {
    const { init } = this.props
    init()
  }
  render () {
    const { active, activate, streams } = this.props

    return (<div className="app">
      <Alert />
      <Notifications />
      <Input />
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
