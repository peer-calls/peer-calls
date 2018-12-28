import PropTypes from 'prop-types'
import React from 'react'
import socket from '../socket.js'
import Input from './Input.js'

export const MessagePropTypes = PropTypes.shape({
  userId: PropTypes.string.isRequired,
  message: PropTypes.string.isRequired,
  timestamp: PropTypes.string.isRequired,
  image: PropTypes.string
})

export default class Chat extends React.PureComponent {
  static propTypes = {
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired,
    videos: PropTypes.object.isRequired,
    notify: PropTypes.func.isRequired,
    sendMessage: PropTypes.func.isRequired,
  }
  scrollToBottom = () => {
    this.chatScroll.scrollTop = this.chatScroll.scrollHeight
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
      <div>
        <div className="drawer-content chat-history"
          ref={div => { this.chatScroll = div }}>

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
            <div className="drawer-empty">
              <span className="drawer-empty-icon icon icon-question_answer" />
              <div className="drawer-empty-message">No Notifications</div>
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
