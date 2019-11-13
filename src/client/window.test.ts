import {
  createObjectURL,
  revokeObjectURL,
  getUserMedia,
  navigator,
  play,
  valueOf,
} from './window'

describe('window', () => {

  describe('getUserMedia', () => {

    class MediaStream {}
    const stream = new MediaStream()
    const constraints = { video: true }

    afterEach(() => {
      delete (navigator as any).mediaDevices
      delete navigator.getUserMedia
      delete (navigator as any).webkitGetUserMedia
    })

    it('calls navigator.mediaDevices.getUserMedia', () => {
      const promise = Promise.resolve(stream);
      (navigator as any).mediaDevices = {
        getUserMedia: jest.fn().mockReturnValue(promise),
      }
      expect(getUserMedia(constraints)).toBe(promise)
    })

    ;['getUserMedia', 'webkitGetUserMedia'].forEach((method) => {
      it(`it calls navigator.${method} as a fallback`, () => {
        (navigator as any)[method] = jest.fn()
        .mockImplementation(
          (constraints, onSuccess, onError) => onSuccess(stream),
        )
        return getUserMedia(constraints)
        .then(s => expect(s).toBe(stream))
      })
    })

    it('throws error when no supported method', async () => {
      let error: Error
      try {
        await getUserMedia(constraints)
      } catch (err) {
        error = err
      }
      expect(error!).toBeTruthy()
      expect(error!.message).toBe('Browser unsupported')
    })

  })

  describe('play', () => {

    let v1: HTMLVideoElement & { play: jest.Mock }
    let v2: HTMLVideoElement & { play: jest.Mock }
    beforeEach(() => {
      v1 = window.document.createElement('video') as any
      v2 = window.document.createElement('video') as any
      window.document.body.appendChild(v1)
      window.document.body.appendChild(v2)
      v1.play = jest.fn()
      v2.play = jest.fn()
    })
    afterEach(() => {
      window.document.body.removeChild(v1)
      window.document.body.removeChild(v2)
    })

    it('gets all videos and plays them', () => {
      play()
      expect(v1.play.mock.calls.length).toBe(1)
      expect(v2.play.mock.calls.length).toBe(1)
    })

    it('does not fail on error', () => {
      v1.play.mockImplementation(() => { throw new Error('test') })
      play()
      expect(v1.play.mock.calls.length).toBe(1)
      expect(v2.play.mock.calls.length).toBe(1)
    })

  })

  describe('navigator', () => {

    it('exposes window.navigator', () => {
      expect(navigator).toBe(window.navigator)
    })

  })

  describe('createObjectURL', () => {

    it('calls window.URL.createObjectURL', () => {
      window.URL.createObjectURL = jest.fn().mockReturnValue('test')
      expect(createObjectURL('bla')).toBe('test')
    })

  })

  describe('createObjectURL', () => {

    it('calls window.URL.revokeObjectURL', () => {
      window.URL.revokeObjectURL = jest.fn()
      expect(revokeObjectURL('bla')).toBe(undefined)
    })

  })

  describe('valueOf', () => {

    let input: HTMLInputElement
    beforeEach(() => {
      input = window.document.createElement('input')
      input.setAttribute('id', 'my-main-id')
      input.value = 'test'
      window.document.body.appendChild(input)
    })
    afterEach(() => {
      window.document.body.removeChild(input)
    })

    it('should return value of input', () => {
      expect(valueOf('my-main-id')).toEqual('test')
    })

    it('does not fail when not found', () => {
      expect(valueOf('my-main-id2')).toEqual(null)
    })

  })

})
