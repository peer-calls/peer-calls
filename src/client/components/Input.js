import PropTypes from 'prop-types'
import React from 'react'
import moment from 'moment'
import socket from '../socket.js'

export default class Input extends React.PureComponent {
  static propTypes = {
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
    if (e.key === 'Enter') {
      e.preventDefault()
      this.submit()
    }
  }
  submit = () => {
    const { notify, sendMessage } = this.props
    const { message } = this.state
    if (message) {
      notify('You: ' + message)
      sendMessage(message)

      const userId = socket.id
      const timestamp = moment().format('ddd, D MMM HH:mm a')
      const payload = { userId, message, timestamp }
      socket.emit('new_message', payload)
    }
    this.setState({ message: '' })
  }
  render () {
    const { message } = this.state
    return (
      <form className="input" onSubmit={this.handleSubmit}>
        <input
          onChange={this.handleChange}
          onKeyPress={this.handleKeyPress}
          placeholder="Enter your message..."
          type="text"
          value={message}
        />
        <input type="submit" value="Send" />
      </form>
    )
  }
}
