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

    if (!ratios.length) {
      return
    }

    // We decide that all windows will have the same y.
    // Since we have the dimensions of all videos, we can easily calculate the
    // areas from just y since we know the aspect ratio:
    //
    //     r = x/y => x = r*y
    //             => y = x/r
    //
    // So if we have N windows, each with different ratios:
    //
    //     x1*y1 + x2*y2 + ... + xn*yn <= X*Y
    //
    //  And we've already said that all ys are the same:
    //
    //     x1*y + x2*y + ... + xn*y <= X*Y
    //
    //     r1*y^2 + r1*y^2 + ... + rn*y^2 <= X*Y
    //
    // Since all numbers are positive, we can easily calculate max y:
    //
    //     y <= sqrt(X*Y / (r1 + r2 + ... rn))
    //
    // We can also invert this equation by calculating the max x:
    //
    //     x*y1 + x*y2 + ... + x*yn <= X*Y
    //
    //     x^2/r1 + x^2/r2 + ... + x^2/rn <= X*Y
    //
    //     x^2(1/r1 + 1/r2 + ... + 1/rn) <= X*Y
    //
    //     x <= sqrt(X*Y / (1/r1 + 1/r2 + ... + 1/rn))
    //
    // But this doesn't take into account if a certain window will have to be
    // split since we only take the areas into account.
    //
    // We need to add an additional constraint for x so we don't exceed the
    // X per row.
    //
    // Additional rules:
    //
    // The maximum width of any window is X
    // The maximum height of any window is Y
    //
    //

    const XY = size.x * size.y
    const sumRatios = ratios.reduce((sum, r) => sum + r, 0)

    // We calculate the maximum possible y.
    const maxY = Math.sqrt(XY / sumRatios)

    // However, we still need to make sure that all the windows fit within
    // the viewport size.
    // const minNumRows = Math.floor(size.y / maxY)
    return maxY
  }
}

export class MultiFrame2 {
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
    // const { ratios, size } = this

    // const numWindows = ratios.length
    // const maxRows = numWindows

    // for (let i = 1; i <= maxRows; i++) {
    //   this.getPermutations(i)
    // }
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

    console.log('permutation', i, permutation, 'maxIndex', maxIndex)

    let last = permutation[permutation.length - 1]

    for(let i = permutation.length - 2; i >= 1; i--) {
      const j = i - 1

      const vi = permutation[i]
      const vj = permutation[j]

      if (vi === vj && vi === last - 1 && i >= maxIndex) {
        const newPerm = permutation.slice()
        newPerm[i]++
        console.log('  new', newPerm, 'maxIndex', j)
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
