import Input from './Input.js'
import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import socket from '../socket.js'

export const MessagePropTypes = PropTypes.shape({
  userId: PropTypes.string.isRequired,
  message: PropTypes.string.isRequired,
  timestamp: PropTypes.string.isRequired,
  image: PropTypes.string
})

export default class Chat extends React.PureComponent {
  static propTypes = {
    visible: PropTypes.bool.isRequired,
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired,
    notify: PropTypes.func.isRequired,
    onClose: PropTypes.func.isRequired,
    sendMessage: PropTypes.func.isRequired,
    videos: PropTypes.object.isRequired
  }
  constructor () {
    super()
    this.chatHistoryRef = React.createRef()
  }
  scrollToBottom = () => {
    const chatHistoryRef = this.chatHistoryRef.current
    chatHistoryRef.scrollTop = chatHistoryRef.scrollHeight
  }
  componentDidMount () {
    this.scrollToBottom()
  }
  componentDidUpdate () {
    this.scrollToBottom()
  }
  render () {
    const { messages, videos, notify, sendMessage } = this.props
    return (
      <div className={classnames('chat-container', {
        show: this.props.visible
      })}>
        <div className="chat-header">
          <div className="chat-close" onClick={this.props.onClose}>
            <div className="button button-icon">
              <span className="icon icon-arrow_forward" />
            </div>
          </div>
          <div className="chat-title">Chat</div>
        </div>
        <div className="chat-history" ref={this.chatHistoryRef}>

          {messages.length ? (
            messages.map((message, i) => (
              <div key={i}>
                {message.userId === socket.id ? (
                  <div className="chat-item chat-item-me">
                    <div className="message">
                      <span className="message-user-name">
                        {message.userId}
                      </span>
                      <span className="icon icon-schedule" />
                      <time className="message-time">{message.timestamp}</time>
                      <p className="message-text">{message.message}</p>
                    </div>
                    {message.image ? (
                      <img className="chat-item-img" src={message.image} />
                    ) : (
                      <span className="chat-item-img icon icon-face" />
                    )}
                  </div>
                ) : (
                  <div className="chat-item chat-item-other">
                    {message.image ? (
                      <img className="chat-item-img" src={message.image} />
                    ) : (
                      <span className="chat-item-img icon icon-face" />
                    )}
                    <div className="message">
                      <span className="message-user-name">
                        {message.userId}
                      </span>
                      <span className="icon icon-schedule" />
                      <time className="message-time">{message.timestamp}</time>
                      <p className="message-text">{message.message}</p>
                    </div>
                  </div>
                )}
              </div>
            ))
          ) : (
            <div className="chat-empty">
              <span className="chat-empty-icon icon icon-question_answer" />
              <div className="chat-empty-message">No Notifications</div>
            </div>
          )}

        </div>

        <Input
          videos={videos}
          notify={notify}
          sendMessage={sendMessage}
        />
      </div>
    )
  }
}
