import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import { ME } from '../constants.js'

export default class Video extends React.Component {
  static propTypes = {
    onClick: PropTypes.func,
    active: PropTypes.bool.isRequired,
    stream: PropTypes.string.isRequired,
    userId: PropTypes.string.isRequired
  }
  handleClick = e => {
    const { onClick, userId } = this.props
    this.play(e)
    onClick(userId)
  }
  play = e => {
    e.preventDefault()
    e.target.play()
  }
  render () {
    const { active, stream, userId } = this.props
    const className = classnames('video-container', { active })
    return (
      <div className={className}>
        <video
          muted={userId === ME}
          onClick={this.handleClick}
          onLoadedMetadata={this.play}
          src={stream}
        />
      </div>
    )
  }
}
