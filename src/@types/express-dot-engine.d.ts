declare module 'express-dot-engine' {
  function render(path: string, options: object, callback: (e: any, rendered: string) => void): void

  interface ExpressDotEngine {
    __express: typeof render
  }

  declare const engine: ExpressDotEngine
  export = engine
}
