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
    toolbarRef: PropTypes.object.isRequired,
    messages: PropTypes.arrayOf(MessagePropTypes).isRequired
  }
  handleCloseChat = e => {
    const { toolbarRef } = this.props
    toolbarRef.chatButton.click()
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
    const { messages } = this.props
    return (
      <div>
        <div className="chat-header">
          <div className="chat-close" onClick={this.handleCloseChat}>
            <div className="button button-icon">
              <span className="icon icon-arrow_forward" />
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
                  <div className="profile-image-component
                    profile-image-component-circle">
                    {message.image ? (
                      <div className="profile-image-component-image">
                        <img src={message.image} />
                      </div>
                    ) : (
                      <div className="profile-image-component-initials">
                        {message.userId.substr(0, 2).toUpperCase()}
                      </div>
                    )}
                  </div>
                </div>
                <div className="chat-item-content">
                  {message.message}
                </div>
              </div>
            ))
          ) : (
            <div className="chat-empty">
              <span className="chat-empty-icon icon icon-question_answer" />
              <div className="chat-empty-message">No Notifications</div>
            </div>
          )}

        </div>
      </div>
    )
  }
}
