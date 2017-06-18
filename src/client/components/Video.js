import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import { ME } from '../constants.js'

export default class Video extends React.Component {
  static propTypes = {
    setActive: PropTypes.func.isRequired,
    active: PropTypes.bool.isRequired,
    stream: PropTypes.string.isRequired,
    userId: PropTypes.string.isRequired
  }
  setActive = e => {
    const { setActive, userId } = this.props
    this.play(e)
    setActive(userId)
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
          onClick={this.setActive}
          onLoadedMetadata={this.play}
          src={stream}
        />
      </div>
    )
  }
}
