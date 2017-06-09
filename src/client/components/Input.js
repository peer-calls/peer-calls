import React from 'react'
import notify from '../action/notify.js'
import peers from '../peer/peers.js'

export default class Input extends React.PureComponent {
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
    e.preventDefault()
    e.key === 'Enter' && this.submit()
  }
  submit = () => {
    const { message } = this.state
    peers.message(message)
    notify.info('You: ' + message)
    this.setState({ message: '' })
  }
  render () {
    const { message } = this.state
    return (
      <form className="input" onSubmit={this.handleSubmit}>
        <input
          onChange={this.handleChange}
          onKeyPress={this.onKeyPress}
          placeholder="Enter your message..."
          type="text"
          value={message}
        />
      </form>
    )
  }
}
