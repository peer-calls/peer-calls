import classnames from 'classnames'
import React from 'react'
import { Message as MessageType } from '../actions/ChatActions'
import { TextMessage } from '../actions/PeerActions'
import Input from './Input'

export interface MessageProps {
  message: MessageType
}

function Message (props: MessageProps) {
  const { message } = props
  return (
    <p className="message-text">
      {message.image && (
        <img src={message.image} width="100%" />
      )}
      {message.message}
    </p>
  )
}

export interface ChatProps {
  visible: boolean
  messages: MessageType[]
  onClose: () => void
  sendMessage: (message: TextMessage) => void
}

export default class Chat extends React.PureComponent<ChatProps> {
  chatHistoryRef = React.createRef<HTMLDivElement>()
  inputRef = React.createRef<Input>()

  scrollToBottom = () => {
    const chatHistoryRef = this.chatHistoryRef.current!
    chatHistoryRef.scrollTop = chatHistoryRef.scrollHeight
  }
  componentDidMount () {
    this.scrollToBottom()
    this.focus()
  }
  componentDidUpdate () {
    this.scrollToBottom()
    this.focus()
  }
  focus() {
    if (this.props.visible) {
      this.inputRef.current?.textArea.current?.focus()
    }
  }
  render () {
    const { messages, sendMessage } = this.props
    return (
      <div className={classnames('chat-container', {
        show: this.props.visible,
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
                {message.userId === 'You' ? (
                  <div className="chat-item chat-item-me">
                    <div className="message">
                      <span className="message-user-name">
                        {message.userId}
                      </span>
                      <span className="icon icon-schedule" />
                      <time className="message-time">{message.timestamp}</time>
                      <Message message={message} />
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
                      <Message message={message} />
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

        <Input ref={this.inputRef} sendMessage={sendMessage} />
      </div>
    )
  }
}
