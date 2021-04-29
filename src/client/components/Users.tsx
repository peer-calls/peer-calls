import React from 'react'
import { Nicknames } from '../reducers/nicknames'
import map from 'lodash/map'

export interface UsersProps {
  nicknames: Nicknames
}

interface UserProps {
  peerId: string
  nickname: string
}

class User extends React.PureComponent<UserProps> {
  render() {
    return (
      <li>
        <label>
          <input type='checkbox' />
          {this.props.nickname}
        </label>
      </li>
    )
  }
}

export default class Users extends React.PureComponent<UsersProps> {
  render() {
    const { nicknames } = this.props
    return (
      <ul className='users'>
        {map(nicknames, (nickname, peerId) => (
          <User
            key={peerId}
            nickname={nickname}
            peerId={peerId}
          />
        ))}
      </ul>
    )
  }
}
