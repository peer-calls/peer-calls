export const createObjectURL = jest.fn()
.mockImplementation(object => 'blob://' + String(object))
export const revokeObjectURL = jest.fn()

export class MediaStream {
  getVideoTracks () {
    return [{
      enabled: true,
    }]
  }
  getAudioTracks () {
    return [{
      enabled: true,
    }]
  }
}
export function getUserMedia () {
  return !(getUserMedia as any).shouldFail
    ? Promise.resolve(getUserMedia.stream)
    : Promise.reject(new Error('test'))
}
getUserMedia.fail = (shouldFail: boolean) => (getUserMedia as any).shouldFail = shouldFail
getUserMedia.stream = new MediaStream()

export const navigator = window.navigator

export const play = jest.fn()

export const valueOf = jest.fn()

export const callId = 'call1234'

export const iceServers = []
