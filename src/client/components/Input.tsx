import React, { ReactEventHandler, ChangeEventHandler, KeyboardEventHandler, MouseEventHandler } from 'react'
import { TextMessage } from '../actions/PeerActions'

export interface InputProps {
  sendMessage: (message: TextMessage) => void
}

export interface InputState {
  message: string
}

export default class Input extends React.PureComponent<InputProps, InputState> {
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
  submit = () => {
    const { sendMessage } = this.props
    const { message } = this.state
    if (message) {
      sendMessage({
        payload: message,
        type: 'text',
      })
      // let image = null

      // // take snapshoot
      // try {
      //   const video = videos[userId]
      //   if (video) {
      //     const canvas = document.createElement('canvas')
      //     canvas.height = video.videoHeight
      //     canvas.width = video.videoWidth
      //     const avatar = canvas.getContext('2d')
      //     avatar.drawImage(video, 0, 0, canvas.width, canvas.height)
      //     image = canvas.toDataURL()
      //   }
      // } catch (e) {}
    }
    this.setState({ message: '' })
  }
  render () {
    const { message } = this.state
    return (
      <form className="chat-controls" onSubmit={this.handleSubmit}>
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
            className="chat-controls-buttons-send" />

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
