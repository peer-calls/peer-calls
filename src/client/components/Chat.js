import PropTypes from 'prop-types'
import React from 'react'
import socket from '../socket.js'

export const MessagePropTypes = PropTypes.shape({
  userId: PropTypes.string.isRequired,
  message: PropTypes.string.isRequired,
  timestamp: PropTypes.string.isRequired
})

export default class Chat extends React.PureComponent {
  static propTypes = {
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired
  }
  hideChat = e => {
    document.getElementById('chat').classList.remove('show')
    document.querySelector('.toolbar .chat').classList.remove('on')
  }
  render () {
    const { messages } = this.props
    return (
      <div>
        <div className="chat-header">
          <div className="chat-close" onClick={this.hideChat}>
            <div className="button button-icon">
              <span className="material-icons">arrow_back</span>
            </div>
          </div>
          <div className="chat-title">Chat</div>
        </div>
        <div className="chat-content">

          {messages.length ? (
            messages.map((message, i) => (
              <div className={message.userId === socket.id ? 'chat-bubble alt' : 'chat-bubble'} key={i}>
                <div className="txt">
                  <p className="name">{message.userId}</p>
                  <p className="message">{message.message}</p>
                  <span className="timestamp">{message.timestamp}</span>
                </div>
                <div className="arrow"></div>
              </div>
            ))
          ) : (
            <div className="chat-empty">
              <div className="chat-empty-icon material-icons">chat</div>
              <div className="chat-empty-message">No Notifications</div>
            </div>
          )}

        </div>
      </div>
    )
  }
}
