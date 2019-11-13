declare module 'screenfull' {
  interface Screenfull {
    enabled: boolean
    toggle: () => void
  }

  declare const screenfull: Screenfull

  export = screenfull
}
