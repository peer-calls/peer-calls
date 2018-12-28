import PropTypes from 'prop-types'
import React from 'react'
import _ from 'underscore'
import * as constants from '../constants.js'
import Alerts, { AlertPropType } from './Alerts.js'
import Toolbar from './Toolbar.js'
import Notifications, { NotificationPropTypes } from './Notifications.js'
import Chat, { MessagePropTypes } from './Chat.js'
import Map from './Map.js'
import Video, { StreamPropType } from './Video.js'

export default class App extends React.PureComponent {
  static propTypes = {
    active: PropTypes.string,
    alerts: PropTypes.arrayOf(AlertPropType).isRequired,
    dismissAlert: PropTypes.func.isRequired,
    init: PropTypes.func.isRequired,
    notifications: PropTypes.objectOf(NotificationPropTypes).isRequired,
    notify: PropTypes.func.isRequired,
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired,
    positions: PropTypes.object.isRequired,
    peers: PropTypes.object.isRequired,
    sendMessage: PropTypes.func.isRequired,
    streams: PropTypes.objectOf(StreamPropType).isRequired,
    toggleActive: PropTypes.func.isRequired
  }
  constructor () {
    super()
    this.state = {
      videos: {},
      isOpenChat: true,
      isOpenMap: false
    }

    this.drawerRef = React.createRef()
  }
  handleCloseDrawer = e => {
    this.toolbarRef.drawerButton.click()
  }
  handleToggleDrawer = e => {
    const { isOpenChat, isOpenMap } = this.state
    this.setState({ isOpenChat: !isOpenChat })
    this.setState({ isOpenMap: !isOpenMap })
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
      messages,
      positions,
      peers,
      sendMessage,
      toggleActive,
      streams
    } = this.props

    const { videos, isOpenChat, isOpenMap } = this.state

    return (
      <div className="app">
        <Toolbar
          drawerRef={this.drawerRef}
          messages={messages}
          stream={streams[constants.ME]}
          ref={node => { this.toolbarRef = node }}
        />
        <Alerts alerts={alerts} dismiss={dismissAlert} />
        <Notifications notifications={notifications} />
        <div className="drawer-container"
          ref={node => { this.drawerRef = node }}
        >
          {!constants.GOOGLE_MAPS_API_KEY || isOpenChat ? (
            <div>
              <div className="drawer-header">
                <div className="drawer-close" onClick={this.handleCloseDrawer}>
                  <span className="icon icon-arrow_forward" />
                </div>
                {constants.GOOGLE_MAPS_API_KEY && (
                  <div className="drawer-button"
                    onClick={this.handleToggleDrawer}>
                    <span className="icon icon-room" />
                  </div>
                )}
                <div className="drawer-title">Chat</div>
              </div>

              <Chat
                messages={messages}
                videos={videos}
                notify={notify}
                sendMessage={sendMessage}
              />
            </div>
          ) : isOpenMap ? (
            <div>
              <div className="drawer-header">
                <div className="drawer-close" onClick={this.handleCloseDrawer}>
                  <span className="icon icon-arrow_forward" />
                </div>
                <div className="drawer-button"
                  onClick={this.handleToggleDrawer}>
                  <span className="icon icon-question_answer" />
                </div>
                <div className="drawer-title">Map</div>
              </div>

              <Map positions={positions} />
            </div>
          ) : null}
        </div>
        <div className="videos">
          <Video
            videos={videos}
            active={active === constants.ME}
            onClick={toggleActive}
            stream={streams[constants.ME]}
            userId={constants.ME}
            muted
          />

          {_.map(peers, (_, userId) => (
            <Video
              active={userId === active}
              key={userId}
              onClick={toggleActive}
              stream={streams[userId]}
              userId={userId}
              videos={videos}
            />
          ))}
        </div>
      </div>
    )
  }
}
