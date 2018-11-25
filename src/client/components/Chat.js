import PropTypes from 'prop-types'
import React from 'react'

export const MessagePropTypes = PropTypes.shape({
  userId: PropTypes.string.isRequired,
  message: PropTypes.string.isRequired,
  timestamp: PropTypes.string.isRequired,
  image: PropTypes.string
})

export default class Chat extends React.PureComponent {
  static propTypes = {
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired
  }
  handleCloseChat = e => {
    document.getElementById('chat').classList.remove('show')
    document.querySelector('.toolbar .chat').classList.remove('on')
  }
  scrollToBottom = () => {
    // this.chatScroll.scrollTop = this.chatScroll.scrollHeight

    const duration = 300
    const start = this.chatScroll.scrollTop
    const end = this.chatScroll.scrollHeight
    const change = end - start
    const increment = 20

    const easeInOut = (currentTime, start, change, duration) => {
      currentTime /= duration / 2
      if (currentTime < 1) {
        return change / 2 * currentTime * currentTime + start
      }
      currentTime -= 1
      return -change / 2 * (currentTime * (currentTime - 2) - 1) + start
    }

    const animate = elapsedTime => {
      elapsedTime += increment
      const position = easeInOut(elapsedTime, start, change, duration)
      this.chatScroll.scrollTop = position
      if (elapsedTime < duration) {
        setTimeout(() => {
          animate(elapsedTime)
        }, increment)
      }
    }

    animate(0)
  }
  componentDidMount () {
    this.scrollToBottom()
  }
  componentDidUpdate () {
    this.scrollToBottom()
  }
  render () {
    const { messages } = this.props
    return (
      <div>
        <div className="chat-header">
          <div className="chat-close" onClick={this.handleCloseChat}>
            <div className="button button-icon">
              <span className="material-icons">arrow_forward</span>
            </div>
          </div>
          <div className="chat-title">Chat</div>
        </div>
        <div className="chat-content" ref={div => { this.chatScroll = div }}>

          {messages.length ? (
            messages.map((message, i) => (
              <div key={i} className="chat-item">
                <div className="chat-item-label" />
                <div className="chat-item-icon">
                  {message.image ? (
                    <div className="profile-image-component
                      profile-image-component-circle">
                      <div className="profile-image-component-image">
                        <img src={message.image} />
                      </div>
                    </div>
                  ) : (
                    <div className="profile-image-component
                      profile-image-component-circle">
                      <div className="profile-image-component-initials">
                        {message.userId.substr(0, 2).toUpperCase()}
                      </div>
                    </div>
                  )}
                </div>
                <div className="chat-item-content">
                  {message.message}
                </div>
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
