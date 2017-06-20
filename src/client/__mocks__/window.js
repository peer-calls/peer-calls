import Promise from 'bluebird'

export const createObjectURL = jest.fn()
.mockImplementation(object => 'blob://' + String(object))
export const revokeObjectURL = jest.fn()

export class MediaStream {}
export function getUserMedia () {
  return !getUserMedia.shouldFail
  ? Promise.resolve(getUserMedia.stream)
  : Promise.reject(new Error('test'))
}
getUserMedia.fail = shouldFail => getUserMedia.shouldFail = shouldFail
getUserMedia.stream = new MediaStream()

export const navigator = window.navigator

export const play = jest.fn()

export const valueOf = jest.fn()

export const callId = 'call1234'

export const iceServers = []
