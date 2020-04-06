export const createObjectURL = jest.fn()
.mockImplementation(object => 'blob://' + String(object))
export const revokeObjectURL = jest.fn()

let count = 0

export class MediaStream {
  id: string
  addTrack = jest.fn()
  removeTrack = jest.fn()
  getTracks = jest.fn().mockReturnValue([{
    stop: jest.fn(),
  }, {
    stop: jest.fn(),
  }])
  getVideoTracks = jest.fn().mockReturnValue([{
    enabled: true,
    stop: jest.fn(),
  }])
  getAudioTracks = jest.fn().mockReturnValue([{
    stop: jest.fn(),
    enabled: true,
  }])

  constructor() {
    this.id = String(++count)
  }
}

export class MediaStreamTrack {}

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
