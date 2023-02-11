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
