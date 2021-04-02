import { Nicknames } from './reducers/nicknames'
import { ME } from './constants'

export function getNickname(nicknames: Nicknames, peerId: string): string {
  const nickname = nicknames[peerId]
  if (nickname) {
    return nickname
  }
  if (peerId === ME) {
    return 'You'
  }
  return peerId
}
