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
export const navigator = window.navigator

;(window as any).navigator.mediaDevices = {}
window.navigator.mediaDevices.enumerateDevices = async () => {
  return []
}
window.navigator.mediaDevices.getUserMedia = async () => {
  return {} as any
}

export const play = jest.fn()

export const valueOf = jest.fn()

export const callId = 'call1234'

export const iceServers = []
