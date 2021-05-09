import map from 'lodash/map'
import React from 'react'
import { connect } from 'react-redux'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { getStreamsByState, StreamProps } from '../selectors'
import { State } from '../store'
import uniqueId from 'lodash/uniqueId'

export interface UsersProps {
  streams: StreamProps[]
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  play: () => void
}

interface UserProps extends StreamProps {
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
  play: () => void
}

class User extends React.PureComponent<UserProps> {
  uniqueId: string
  constructor(props: UserProps) {
    super(props)
    this.uniqueId = uniqueId('user-')
  }
  handleChange = () => {
    const { peerId, stream } = this.props
    const streamId = stream && stream.streamId

    this.props.onMinimizeToggle({
      peerId,
      streamId,
    })
  }
  render() {
    return (
      <li>
        <label htmlFor={this.uniqueId}>
          <input
            id={this.uniqueId}
            type='checkbox'
            checked={this.props.windowState !== 'minimized' }
            onChange={this.handleChange}
          />
          {this.props.nickname}
        </label>
      </li>
    )
  }
}

class Users extends React.PureComponent<UsersProps> {
  render() {
    const { onMinimizeToggle, play, streams } = this.props

    return (
      <div className='users'>
        <ul className='users-list'>
          {map(streams, (stream) => (
            <User
              {...stream}
              key={stream.key}
              onMinimizeToggle={onMinimizeToggle}
              play={play}
            />
          ))}
        </ul>
        <div></div> {/*necessary for flex to stretch */}
      </div>
    )
  }
}

function mapStateToProps(state: State) {
  const { all } = getStreamsByState(state)

  return {
    streams: all,
  }
}

export default connect(mapStateToProps)(Users)
