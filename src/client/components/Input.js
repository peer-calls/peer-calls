import PropTypes from 'prop-types'
import React from 'react'
import peers from '../peer/peers.js'

export default class Input extends React.Component {
  static propTypes = {
    notify: PropTypes.func.isRequired
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
    const { notify } = this.props
    const { message } = this.state
    peers.message(message)
    notify('You: ' + message)
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
      </form>
    )
  }
}
