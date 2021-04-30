import map from 'lodash/map'
import React from 'react'
import { connect } from 'react-redux'
import { MinimizeTogglePayload } from '../actions/StreamActions'
import { getStreamsByState, StreamProps } from '../selectors'

import { State } from '../store'

export interface UsersProps {
  streams: StreamProps[]
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
}

interface UserProps extends StreamProps {
  onMinimizeToggle: (payload: MinimizeTogglePayload) => void
}

class User extends React.PureComponent<UserProps> {
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
        <label>
          <input
            type='checkbox'
            checked={this.props.windowState !== 'minimized' }
            onClick={this.handleChange}
          />
          {this.props.nickname}
        </label>
      </li>
    )
  }
}

class Users extends React.PureComponent<UsersProps> {
  render() {
    const { onMinimizeToggle, streams } = this.props
    return (
      <ul className='users'>
        {map(streams, (stream) => (
          <User
            {...stream}
            key={stream.key}
            onMinimizeToggle={onMinimizeToggle}
          />
        ))}
      </ul>
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
