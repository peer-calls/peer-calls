export interface Dim {
  x: number
  y: number
}

export class Frame {
  private numWindows = 0
  private size: Dim = {
    x: 0,
    y: 0,
  }
  private lastCalc?: Dim

  constructor(private aspectRatio: number) {
    if (aspectRatio <= 0) {
      throw new Error('aspectRatio must be greater than zero: ' + aspectRatio)
    }
  }

  public setAspectRatio(aspectRatio: number) {
    if (aspectRatio <= 0) {
      throw new Error('aspectRatio must be greater than zero: ' + aspectRatio)
    }

    if (this.aspectRatio === aspectRatio) {
      return
    }

    this.aspectRatio = aspectRatio
    this.lastCalc = undefined
  }

  public setNumWindows(numWindows: number) {
    if (numWindows < 0) {
      throw new Error('numWindows must zero or greater: ' + numWindows)
    }

    if (this.numWindows === numWindows) {
      return
    }

    this.numWindows = Math.floor(numWindows)
    this.lastCalc = undefined
  }

  public setSize(size: Dim) {
    if (this.size.x === size.x && this.size.y === size.y) {
      return
    }

    this.size = size
    this.lastCalc = undefined
  }

  public needsCalc(): boolean {
    return this.lastCalc === undefined
  }

  public calcSize(): Dim {
    if (this.lastCalc) {
      return this.lastCalc
    }

    const {x: X, y: Y} = this.size
    const {aspectRatio, numWindows} = this

    if (X === 0 || Y === 0) {
      this.lastCalc = {x: 0, y: 0}
      return this.lastCalc
    }

    interface DimWithArea {
      dim: Dim
      area: number
    }

    let itemWithLargestArea: DimWithArea = {
      dim: {x: 0, y: 0},
      area: 0,
    }

    // We go in a loop over the total number of windows and try to populate
    // a grid. Then we decide which window size has the highest are and use
    // that.
    for (let i = 1; i <= numWindows; i++) {
      const nRow = i
      const nCol = Math.ceil(numWindows / nRow)

      const maxX = X / nRow
      const maxY = Y / nCol

      const ratio = maxX / maxY

      if (ratio === aspectRatio) {
        // We can't get any better than this, whole area can be completely
        // filled.
        this.lastCalc = {x: maxX, y: maxY}
        return this.lastCalc
      }

      let x = 0
      let y = 0

      if (ratio > aspectRatio) {
        // Need to reduce x
        y = maxY
        x = y * aspectRatio
      } else {
        // Need to reduce y
        x = maxX
        y = x / aspectRatio
      }

      const d: DimWithArea = {
        dim: {x, y},
        area: x*y,
      }

      if (d.area > itemWithLargestArea.area) {
        itemWithLargestArea = d
      }
    }

    this.lastCalc = {
      x: Math.round(itemWithLargestArea.dim.x * 100) / 100,
      y: Math.round(itemWithLargestArea.dim.y * 100) / 100,
    }

    return this.lastCalc
  }
}

export class MultiFrame {
  private size?: Dim
  private ratios: number[] = []

  public setSize(size: Dim) {
    this.size = size
    this.calc()
  }

  public setRatios(ratios: number[]) {
    this.ratios = ratios
    this.calc()
  }

  calc() {
    const { ratios, size } = this

    if (!size) {
      return
    }

    const numWindows = ratios.length
    // const maxRows = numWindows

    const maxX = size.x
    const maxY = size.y

    const choice = {
      y: 0,
      area: 0,
    }

    for (let numRows = 1; numRows <= numWindows; numRows++) {
      const permutations = getPermutations({
        numWindows,
        numRows,
      })

      const maxYPerRow = maxY / numRows

      permutations.forEach(permutation => {
        let lastRowIndex = -1
        let currentRatio = 0
        const rowRatios = permutation
        .reduce((rr, rowIndex, windowIndex) => {
          if (rowIndex !== lastRowIndex && currentRatio > 0 ) {
              rr.push(currentRatio)
              currentRatio = 0
          }
          currentRatio += ratios[windowIndex]
          lastRowIndex = rowIndex

          return rr
        }, [] as number[])

        if (currentRatio > 0) {
          rowRatios.push(currentRatio)
        }

        const maxRatio = rowRatios.reduce((cur, ratio) => {
          if (ratio > cur) {
            cur = ratio
          }

          return cur
        }, 0)

        let y = maxYPerRow

        const x = maxRatio * y

        if (x > maxX) {
          y = maxX / maxRatio
        }

        const area = ratios.reduce((total, r) => {
          // x/y = r
          // area = x * y
          // area = r * y * y
          return total + r * y * y
        }, 0)

        if (area > choice.area) {
          choice.y = y
          choice.area = area
        }
      })
    }

    return { y: choice.y }
  }

}

export interface getPermutationsParams {
  numWindows: number
  numRows: number
}

export function getPermutations(params: getPermutationsParams) {
  const { numWindows, numRows } = params

  if (numWindows === 0) {
    return []
  }

  if (numWindows < 0) {
    throw new Error('numWindows cannot be negative')
  }

  if (numRows < 0) {
    throw new Error('numRows cannot be negative')
  }

  if (numWindows - numRows < 0) {
    throw new Error('more rows than windows')
  }

  let row = numRows

  const init: number[] = []

  for (let i = 0; i < numWindows; i++) {
    if (row > 0) {
      row--
    }

    init.push(row)
  }

  init.reverse()

  interface Permutation {
    values: number[]
    maxIndex: number
  }

  const permutations: Permutation[] = [{
    values: init,
    maxIndex: 0,
  }]

  for (let i = 0; i < permutations.length; i++) {
    const p = permutations[i]

    const permutation = p.values
    const maxIndex = p.maxIndex

    let last = permutation[permutation.length - 1]

    for(let i = permutation.length - 2; i >= 1; i--) {
      const j = i - 1

      const vi = permutation[i]
      const vj = permutation[j]

      if (vi === vj && vi === last - 1 && i >= maxIndex) {
        const newPerm = permutation.slice()
        newPerm[i]++
        permutations.push({
          values: newPerm,
          maxIndex: j,
        })
      }

      last = vi
    }
  }

  return permutations.map(p => p.values)
}
