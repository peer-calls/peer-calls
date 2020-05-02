import React, { ReactEventHandler, ChangeEventHandler, KeyboardEventHandler, MouseEventHandler } from 'react'
import { Message } from '../actions/PeerActions'

export interface InputProps {
  sendFile: (file: File) => void
  sendMessage: (message: Message) => void
}

export interface InputState {
  message: string
}

const hidden = {
  display: 'none',
}

export default class Input extends React.PureComponent<InputProps, InputState> {
  file = React.createRef<HTMLInputElement>()
  textArea = React.createRef<HTMLTextAreaElement>()
  state = {
    message: '',
  }
  handleChange: ChangeEventHandler<HTMLTextAreaElement> = event => {
    this.setState({
      message: event.target.value,
    })
  }
  handleSubmit: ReactEventHandler<HTMLFormElement> = e => {
    e.preventDefault()
    this.submit()
  }
  handleKeyPress: KeyboardEventHandler<HTMLTextAreaElement> = e => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      this.submit()
    }
  }
  handleSmileClick: MouseEventHandler<HTMLElement> = event => {
    this.setState({
      message: this.textArea.current!.value + event.currentTarget.innerHTML,
    })
  }
  handleSelectFiles = (event: React.ChangeEvent<HTMLInputElement>) => {
    Array.from(event.target!.files!)
    .forEach((file) =>
      this.props.sendFile(file),
    )
  }
  submit = () => {
    const { sendMessage } = this.props
    const { message } = this.state
    if (message) {
      sendMessage({
        payload: message,
        type: 'text',
      })
    }
    this.setState({ message: '' })
  }
  handleSendFile = () => {
    this.file.current!.click()
  }
  render () {
    const { message } = this.state
    return (
      <form className="chat-controls" onSubmit={this.handleSubmit}>
        <input
          style={hidden}
          type='file'
          multiple
          ref={this.file}
          onChange={this.handleSelectFiles}
        />
        <textarea
          className="chat-controls-textarea"
          onChange={this.handleChange}
          onKeyPress={this.handleKeyPress}
          placeholder="Type a message"
          value={message}
          ref={this.textArea}
        />
        <div className="chat-controls-buttons">
          <input type="submit" value="Send"
            className="chat-controls-buttons-send"
          />
          <input
            type="submit"
            value="Send File"
            className="chat-controls-buttons-send-file"
            onClick={this.handleSendFile}
          />

          <div className="chat-controls-buttons-wrapper">
            <div className="emoji">
              <div className="chat-controls-buttons-smiles">
                <span className="icon icon-sentiment_satisfied" />
                <div className="chat-controls-buttons-smiles-menu">
                  <div className="chat-controls-buttons-smile"
                    onClick={this.handleSmileClick}>😑</div>
                  <div className="chat-controls-buttons-smile"
                    onClick={this.handleSmileClick}>😕</div>
                  <div className="chat-controls-buttons-smile"
                    onClick={this.handleSmileClick}>😊</div>
                  <div className="chat-controls-buttons-smile"
                    onClick={this.handleSmileClick}>😎</div>
                  <div className="chat-controls-buttons-smile"
                    onClick={this.handleSmileClick}>💪</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </form>
    )
  }
}
