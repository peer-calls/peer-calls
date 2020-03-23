export const createObjectURL = jest.fn()
.mockImplementation(object => 'blob://' + String(object))
export const revokeObjectURL = jest.fn()

export class MediaStream {
  getTracks() {
    return [{
      stop: jest.fn(),
    }, {
      stop: jest.fn(),
    }]
  }
  getVideoTracks () {
    return [{
      enabled: true,
      stop: jest.fn(),
    }]
  }
  getAudioTracks () {
    return [{
      stop: jest.fn(),
      enabled: true,
    }]
  }
}
export const navigator = window.navigator

;(window as any).navigator.mediaDevices = {}
window.navigator.mediaDevices.enumerateDevices = async () => {
  return []
}
window.navigator.mediaDevices.getUserMedia = async () => {
  return {} as any
}
(window.navigator.mediaDevices as any).getDisplayMedia = async () => {
  return {} as any
}

export const play = jest.fn()

export const valueOf = jest.fn()

export const callId = 'call1234'

export const userId = 'user1234'

export const nickname = 'nick1234'

export const iceServers = []
