import classnames from 'classnames'
import React from 'react'
import { Message as ChatMessage } from '../actions/ChatActions'
import { Message } from '../actions/PeerActions'
import { Nicknames } from '../reducers/nicknames'
import Input from './Input'
import { ME } from '../constants'
import { getNickname } from '../nickname'
import { MdClose, MdFace, MdQuestionAnswer } from 'react-icons/md'

export interface MessageProps {
  message: ChatMessage
}

function MessageEntry (props: MessageProps) {
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
  messages: ChatMessage[]
  nicknames: Nicknames
  onClose: () => void
  sendFile: (file: File) => void
  sendMessage: (message: Message) => void
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
    const { messages, sendFile, sendMessage } = this.props
    return (
      <div className={classnames('chat-container', {
        show: this.props.visible,
      })}>
        <div className="chat-header">
          <div className="chat-close" onClick={this.props.onClose}>
            <MdClose />
          </div>
          <div className="chat-title">Chat</div>
        </div>
        <div className="chat-history" ref={this.chatHistoryRef}>

          {messages.length ? (
            messages.map((message, i) => (
              <div key={i}>
                {message.userId === ME ? (
                  <div className="chat-item chat-item-me">
                    <div className="message">
                      <span className="message-user-name">
                        {getNickname(this.props.nicknames, message.userId)}
                      </span>
                      <time className="message-time">{message.timestamp}</time>
                      <MessageEntry message={message} />
                    </div>
                    <span className="chat-item-img">
                      <MdFace />
                    </span>
                  </div>
                ) : (
                  <div className="chat-item chat-item-other">
                    <span className="chat-item-img">
                      <MdFace />
                    </span>
                    <div className="message">
                      <span className="message-user-name">
                        {getNickname(this.props.nicknames, message.userId)}
                      </span>
                      <time className="message-time">{message.timestamp}</time>
                      <MessageEntry message={message} />
                    </div>
                  </div>
                )}
              </div>
            ))
          ) : (
            <div className="chat-empty">
              <span className="chat-empty-icon">
                <MdQuestionAnswer />
              </span>
              <div className="chat-empty-message">No Notifications</div>
            </div>
          )}

        </div>

        <Input
          ref={this.inputRef}
          sendMessage={sendMessage}
          sendFile={sendFile}
        />
      </div>
    )
  }
}
