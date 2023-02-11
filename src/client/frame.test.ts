import { Dim, Frame } from './frame'

describe('Frame', () => {
  describe('calc', () => {
    function calc(f: Frame, x: number, y: number, n: number): Dim {
      f.setSize({x, y})
      f.setNumWindows(n)
      return f.calcSize()
    }

    it('1280x720', () => {
      const f = new Frame(16/9)
      expect(calc(f, 1280, 720, 1)).toEqual({x: 1280, y: 720})
      expect(calc(f, 1280, 720, 2)).toEqual({x: 640, y: 360})
      expect(calc(f, 1280, 720, 3)).toEqual({x: 640, y: 360})
      expect(calc(f, 1280, 720, 4)).toEqual({x: 640, y: 360})
      expect(calc(f, 1280, 720, 5)).toEqual({x: 426.67, y: 240})
    })

    it('720x1280', () => {
      const f = new Frame(16/9)
      expect(calc(f, 720, 1280, 1)).toEqual({x: 720, y: 405})
      expect(calc(f, 720, 1280, 2)).toEqual({x: 720, y: 405})
      expect(calc(f, 720, 1280, 3)).toEqual({x: 720, y: 405})
      expect(calc(f, 720, 1280, 4)).toEqual({x: 568.89, y: 320})
      expect(calc(f, 720, 1280, 5)).toEqual({x: 455.11, y: 256})
    })
  })
})
