import { Dim, Frame, getPermutations } from './frame'

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

describe('getPermutations', () => {
  it('returns [] for nw=0 nr=0', () => {
    expect(getPermutations({numWindows: 0, numRows: 0})).toEqual([])
  })

  it('returns [0] for nw=1 nr=1', () => {
    expect(getPermutations({numWindows: 1, numRows: 1})).toEqual([[0]])
  })

  it('returns [0,0] for nw=2 nr=1', () => {
    expect(getPermutations({numWindows: 2, numRows: 1})).toEqual([[0, 0]])
  })

  it('returns [0,0,0] for nw=3 nr=1', () => {
    expect(getPermutations({numWindows: 3, numRows: 1})).toEqual([[0, 0, 0]])
  })

  it('returns permutations for nw=2 nr=2', () => {
    expect(getPermutations({numWindows: 2, numRows: 2})).toEqual([[0, 1]])
  })

  it('returns permutations for nw=3 nr=2', () => {
    expect(getPermutations({numWindows: 3, numRows: 2})).toEqual([
      [0, 0, 1],
      [0, 1, 1],
    ])
  })

  it('returns permutations for nw=3 nr=3', () => {
    expect(getPermutations({numWindows: 3, numRows: 3})).toEqual([
      [0, 1, 2],
    ])
  })

  it('returns permutations for nw=4 nr=1', () => {
    expect(getPermutations({numWindows: 4, numRows: 1})).toEqual([
      [0, 0, 0, 0],
    ])
  })

  it('returns permutations for nw=4 nr=2', () => {
    expect(getPermutations({numWindows: 4, numRows: 2})).toEqual([
      [0, 0, 0, 1],
      [0, 0, 1, 1],
      [0, 1, 1, 1],
    ])
  })

  it('returns permutations for nw=4 nr=3', () => {
    expect(getPermutations({numWindows: 4, numRows: 3})).toEqual([
      [0, 0, 1, 2],
      [0, 1, 1, 2],
      [0, 1, 2, 2],
    ])
  })

  it('returns permutations for nw=4 nr=4', () => {
    expect(getPermutations({numWindows: 4, numRows: 4})).toEqual([
      [0, 1, 2, 3],
    ])
  })

  it('returns permutations for nw=5 nr=1', () => {
    expect(getPermutations({numWindows: 5, numRows: 1})).toEqual([
      [0, 0, 0, 0, 0],
    ])
  })

  it('returns permutations for nw=5 nr=2', () => {
    expect(getPermutations({numWindows: 5, numRows: 2})).toEqual([
      [0, 0, 0, 0, 1],
      [0, 0, 0, 1, 1],
      [0, 0, 1, 1, 1],
      [0, 1, 1, 1, 1],
    ])
  })

  it('returns permutations for nw=5 nr=3', () => {
    expect(getPermutations({numWindows: 5, numRows: 3})).toEqual([
      [ 0, 0, 0, 1, 2 ],
      [ 0, 0, 1, 1, 2 ],
      [ 0, 0, 1, 2, 2 ],
      [ 0, 1, 1, 1, 2 ],
      [ 0, 1, 1, 2, 2 ],
      [ 0, 1, 2, 2, 2 ],
    ])
  })

  it('returns permutations for nw=5 nr=4', () => {
    expect(getPermutations({numWindows: 5, numRows: 4})).toEqual([
      [0, 0, 1, 2, 3],
      [0, 1, 1, 2, 3],
      [0, 1, 2, 2, 3],
      [0, 1, 2, 3, 3],
    ])
  })

  it('returns permutations for nw=5 nr=5', () => {
    expect(getPermutations({numWindows: 5, numRows: 5})).toEqual([
      [0, 1, 2, 3, 4],
    ])
  })

  it('returns permutations for nw=6 nr=1', () => {
    expect(getPermutations({numWindows: 6, numRows: 1})).toEqual([
      [0, 0, 0, 0, 0, 0],
    ])
  })

  it('returns permutations for nw=6 nr=2', () => {
    expect(getPermutations({numWindows: 6, numRows: 2})).toEqual([
      [0, 0, 0, 0, 0, 1],
      [0, 0, 0, 0, 1, 1],
      [0, 0, 0, 1, 1, 1],
      [0, 0, 1, 1, 1, 1],
      [0, 1, 1, 1, 1, 1],
    ])
  })

  it('returns permutations for nw=6 nr=3', () => {
    expect(getPermutations({numWindows: 6, numRows: 3})).toEqual([
      [ 0, 0, 0, 0, 1, 2 ],
      [ 0, 0, 0, 1, 1, 2 ],
      [ 0, 0, 0, 1, 2, 2 ],
      [ 0, 0, 1, 1, 1, 2 ],
      [ 0, 0, 1, 1, 2, 2 ],
      [ 0, 1, 1, 1, 1, 2 ],
      [ 0, 0, 1, 2, 2, 2 ],
      [ 0, 1, 1, 1, 2, 2 ],
      [ 0, 1, 1, 2, 2, 2 ],
      [ 0, 1, 2, 2, 2, 2 ],
    ])
  })

  it('returns permutations for nw=6 nr=4', () => {
    expect(getPermutations({numWindows: 6, numRows: 4})).toEqual([
      [ 0, 0, 0, 1, 2, 3 ],
      [ 0, 0, 1, 1, 2, 3 ],
      [ 0, 0, 1, 2, 2, 3 ],
      [ 0, 1, 1, 1, 2, 3 ],
      [ 0, 0, 1, 2, 3, 3 ],
      [ 0, 1, 1, 2, 2, 3 ],
      [ 0, 1, 1, 2, 3, 3 ],
      [ 0, 1, 2, 2, 2, 3 ],
      [ 0, 1, 2, 2, 3, 3 ],
      [ 0, 1, 2, 3, 3, 3 ],
    ])
  })
  it('returns permutations for nw=6 nr=4', () => {
    expect(getPermutations({numWindows: 6, numRows: 4})).toEqual([
      [ 0, 0, 0, 1, 2, 3 ],
      [ 0, 0, 1, 1, 2, 3 ],
      [ 0, 0, 1, 2, 2, 3 ],
      [ 0, 1, 1, 1, 2, 3 ],
      [ 0, 0, 1, 2, 3, 3 ],
      [ 0, 1, 1, 2, 2, 3 ],
      [ 0, 1, 1, 2, 3, 3 ],
      [ 0, 1, 2, 2, 2, 3 ],
      [ 0, 1, 2, 2, 3, 3 ],
      [ 0, 1, 2, 3, 3, 3 ],
    ])
  })
})
