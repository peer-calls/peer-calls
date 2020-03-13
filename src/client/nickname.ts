import { Nicknames } from './reducers/nicknames'
import { ME } from './constants'

export function getNickname(nicknames: Nicknames, userId: string): string {
  const nickname = nicknames[userId]
  if (nickname) {
    return nickname
  }
  if (userId === ME) {
    return 'You'
  }
  return userId
}
