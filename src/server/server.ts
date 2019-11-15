import { readFileSync } from 'fs'
import { resolve, join } from 'path'
import { Config } from './config'

const projectRoot = resolve(join(__dirname, '../..'))

const readFile = (file: string) => readFileSync(resolve(projectRoot, file))

export function createServer (config: Config, app: Express.Application) {
  if (config.ssl) {
    const key = readFile(config.ssl.key)
    const cert = readFile(config.ssl.cert)
    return require('https').createServer({ key, cert }, app)
  }
  return require('http').createServer(app)
}
