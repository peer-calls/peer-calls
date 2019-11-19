/* eslint @typescript-eslint/no-explicit-any: 0 */
import { readFileSync, statSync } from 'fs'
import { resolve, join } from 'path'
import { safeLoad } from 'js-yaml'
import _debug from 'debug'

const debug = _debug('peercalls:config')

const isObject = (value: unknown) => value !== null && typeof value === 'object'

export class ReadConfig {
  constructor(protected readonly config: any) {}

  get(key: string, defaultValue?: any) {
    let value = this.config
    try {
      key.split('.').forEach(k => {
        if (!Object.prototype.hasOwnProperty.call(value, k)) {
          throw new Error(`Property "${k}" from "${key}" does not exist`)
        }
        value = value[k]
      })
    } catch (err) {
      if (arguments.length === 2) {
        return defaultValue
      } else {
        throw err
      }
    }
    return value
  }

  has(key: string) {
    let c = this.config
    return key.split('.').every(k => {
      const has = Object.prototype.hasOwnProperty.call(c, k)
      if (has) {
        c = c[k]
      }
      return has
    })
  }

  value() {
    return this.config
  }

}

function readConfigFile(filename: string): any {
  return safeLoad(readFileSync(filename, 'utf8'))
}

export function mergeConfig(source: any, destination: any): any {
  const stack = [{src: source, dest: destination}]

  while (stack.length) {
    const {src, dest} = stack.pop()!
    const keys = Object.keys(src)

    keys.forEach(key => {
      const value = src[key]
      if (isObject(value) && !Array.isArray(value)) {
        if (
          !Object.prototype.hasOwnProperty.call(dest, key) ||
          Array.isArray(dest[key]) ||
          !isObject(dest[key])
        ) {
          dest[key] = {}
        }
        stack.push({src: value, dest: dest[key]})
        return
      }
      dest[key] = value
    })
  }

  return destination
}

export function findPackageRoot(path = __dirname): string {
  path = resolve(path)
  let lastPath: undefined | string
  while (lastPath !== path) {
    const file = join(path, 'package.json')
    try {
      const result = statSync(file)
      if (result.isFile()) {
        return path
      }
    } catch (err) {
      // ignore error
    }
    lastPath = path
    path = join(path, '..')
  }
  throw new Error('No package.json found')
}

export function getAllConfigFilesInDirectory(
  dir: string,
  environment: string | undefined,
): string[] {
  const files: string[] = [join(dir, 'default.yaml')]
  if (environment) {
    files.push(join(dir, environment + '.yaml'))
  }
  files.push(join(dir, 'local.yaml'))
  return files
}

export function getAllConfigLocations(
  packageDir: string,
  localDir: string,
  environment: string | undefined,
  extraConfigFile?: string,
): string[] {
  const locations: string[] = [
    ...getAllConfigFilesInDirectory(packageDir, environment),
  ]
  if (localDir !== packageDir) {
    locations.push(...getAllConfigFilesInDirectory(localDir, environment))
  }
  if (extraConfigFile && locations.every(loc => loc !== extraConfigFile)) {
    locations.push(resolve(extraConfigFile))
  }
  return locations
}

export function toSnakeCase(string: string) {
  const value = string.split('_')
  .map(item => item[0].toUpperCase() + item.slice(1).toLowerCase())
  .join('')
  return value[0].toLowerCase() + value.slice(1)
}

export function readConfig(
  env = process.env,
  packageDir = join(findPackageRoot(), 'config'),
  localDir = join(process.cwd(), 'config'),
  extraConfigFile?: string,
) {
  const locations = getAllConfigLocations(
    packageDir, localDir, env.NODE_ENV, extraConfigFile)

  const readFiles = locations
  .map(location => {
    try {
      const result = readConfigFile(location)
      debug('Read config file: %s', location)
      return result
    } catch (err) {
      if (!/ENOENT/.test(err.message)) {
        throw err
      }
      return undefined
    }
  })
  .filter(item => item !== undefined)

  const config = readFiles
  .reduce((merged, config) => mergeConfig(config, merged), {})

  const envConfig: any = {}
  Object.keys(env)
  .filter(key => key.startsWith('PEERCALLS__'))
  .forEach(key => {
    const value = env[key]!
    key = key.slice('PEERCALLS__'.length)
    let cfg = envConfig
    const keys = key.split('__').map(toSnakeCase)
    const lastKey = keys[keys.length - 1]
    keys
    .slice(0, keys.length - 1)
    .forEach(shortKey => {
      cfg = cfg[shortKey] = cfg[shortKey] || {}
    })

    try {
      cfg[lastKey] = JSON.parse(value)
    } catch (err) {
      cfg[lastKey] = value
    }
  })

  const configWithEnv = mergeConfig(envConfig, config)
  debug('Read configuration: %j', configWithEnv)
  return new ReadConfig(configWithEnv)
}
