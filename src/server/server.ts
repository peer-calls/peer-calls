import { readFileSync } from 'fs'
import { resolve, join } from 'path'
import { Config } from './config'
import { createServer as createHTTPSServer } from 'https'
import { createServer as createHTTPServer, RequestListener } from 'http'

const projectRoot = resolve(join(__dirname, '../..'))

const readFile = (file: string) => readFileSync(resolve(projectRoot, file))

export function createServer (config: Config, app: RequestListener) {
  if (config.ssl) {
    const key = readFile(config.ssl.key)
    const cert = readFile(config.ssl.cert)
    return createHTTPSServer({ key, cert }, app)
  }
  return createHTTPServer(app)
}
