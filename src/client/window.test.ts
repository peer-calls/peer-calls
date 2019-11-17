import { createObjectURL, revokeObjectURL, valueOf } from './window'

describe('window', () => {

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
