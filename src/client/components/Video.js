import PropTypes from 'prop-types'
import React from 'react'
import classnames from 'classnames'
import { ME } from '../constants.js'

export const StreamPropType = PropTypes.shape({
  userId: PropTypes.string.isRequired,
  stream: PropTypes.instanceOf(ArrayBuffer).isRequired,
  url: PropTypes.string.isRequired
})

export default class Video extends React.PureComponent {
  static propTypes = {
    activate: PropTypes.func.isRequired,
    active: PropTypes.string.required,
    stream: StreamPropType.isRequired
  }
  activate = e => {
    const { activate, stream: { userId } } = this.props
    this.play(e)
    activate(userId)
  }
  play = e => {
    e.preventDefault()
    e.target.play()
  }
  render () {
    const { active, stream: { userId, url } } = this.props
    const className = classnames('video-container', { active })
    return (
      <div className={className}>
        <video
          muted={userId === ME}
          onClick={this.activate}
          onLoadedMetadata={this.play}
          src={url}
        />
      </div>
    )
  }
}
