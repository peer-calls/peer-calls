const React = require('react')
const peers = require('../peer/peers.js')
const notify = require('../action/notify.js')

const Input = React.createClass({
  getInitialState () {
    return {
      visible: false,
      message: ''
    }
  },
  handleChange (e) {
    this.setState({
      message: e.target.value
    })
  },
  handleSubmit (e) {
    e.preventDefault()
    const { message } = this.state
    peers.message(message)
    notify.info('You: ' + message)
    this.setState({ message: '' })
  },
  render () {
    const { message } = this.state
    return (
      <form className='input' onSubmit={this.handleSubmit}>
        <input
          onChange={this.handleChange}
          placeholder='Enter your message...'
          type='text'
          value={message}
        />
      </form>
    )
  }
})

module.exports = Input
