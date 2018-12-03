import PropTypes from 'prop-types'
import React from 'react'
import socket from '../socket.js'

export default class Input extends React.PureComponent {
  static propTypes = {
    videos: PropTypes.object.isRequired,
    notify: PropTypes.func.isRequired,
    sendMessage: PropTypes.func.isRequired
  }
  constructor () {
    super()
    this.state = {
      message: ''
    }
  }
  handleChange = e => {
    this.setState({
      message: e.target.value
    })
  }
  handleSubmit = e => {
    e.preventDefault()
    this.submit()
  }
  handleKeyPress = e => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      this.submit()
    }
  }
  submit = () => {
    const { videos, notify, sendMessage } = this.props
    const { message } = this.state
    if (message) {
      notify('You: ' + message)
      sendMessage(message)

      const userId = socket.id
      const timestamp = new Date().toLocaleString('en-US', {
        hour: 'numeric',
        minute: 'numeric',
        hour12: false
      })
      let image = null

      // take snapshoot
      try {
        const video = videos[userId]
        if (video) {
          const canvas = document.createElement('canvas')
          canvas.height = video.videoHeight
          canvas.width = video.videoWidth
          const avatar = canvas.getContext('2d')
          avatar.drawImage(video, 0, 0, canvas.width, canvas.height)
          image = canvas.toDataURL()
        }
      } catch (e) {}

      const payload = { userId, message, timestamp, image }
      socket.emit('new_message', payload)
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
        />
        <div className="chat-controls-buttons">
          <input type="submit" value="Send"
            className="chat-controls-buttons-send" />
          <div className="chat-controls-buttons-wrapper" />
        </div>
      </form>
    )
  }
}
