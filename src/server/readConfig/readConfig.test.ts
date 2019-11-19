import { ReadConfig, getAllConfigLocations, getAllConfigFilesInDirectory, findPackageRoot, mergeConfig, readConfig, toSnakeCase } from './readConfig'
import { existsSync, mkdirSync, rmdirSync } from 'fs'
import { join } from 'path'

describe('Config', () => {

  const config = new ReadConfig({
    a: 1,
    b: {
      c: 'test1',
      d: 'test2',
    },
  })

  describe('get', () => {
    it('reads config values recursively', () => {
      expect(config.get('a')).toBe(1)
      expect(config.get('b')).toEqual({c: 'test1', d: 'test2'})
      expect(config.get('b.c')).toBe('test1')
      expect(config.get('b.d')).toBe('test2')
    })
    it('throws an error when key does not exist', () => {
      expect(() => config.get('e')).toThrowError(/does not exist/)
      expect(() => config.get('e.f')).toThrowError(/does not exist/)
    })
    it('returns a default value when provided', () => {
      expect(config.get('b.c', 'test')).toBe('test1')
      expect(config.get('e', 1)).toBe(1)
      expect(config.get('e.f', 2)).toBe(2)
    })
  })

  describe('has', () => {
    it('returns true when config property exists, false otherwise', () => {
      expect(config.has('a')).toBe(true)
      expect(config.has('b')).toBe(true)
      expect(config.has('b.c')).toBe(true)
      expect(config.has('b.d')).toBe(true)
      expect(config.has('e')).toBe(false)
      expect(config.has('e.f')).toBe(false)
    })
  })

})

describe('findPackageRoot', () => {
  const dir = join(__dirname, 'package.json')
  beforeEach(() => {
    if (existsSync(dir)) {
      rmdirSync(dir)
    }
    mkdirSync(dir)
  })
  afterEach(() => {
    if (existsSync(dir)) {
      rmdirSync(dir)
    }
  })
  it('finds package root folder', () => {
    expect(findPackageRoot()).toEqual(jasmine.any(String))
  })
  it('finds package root folder', () => {
    expect(findPackageRoot(__dirname)).toEqual(jasmine.any(String))
  })
  it('throws an error when not found', () => {
    expect(() => findPackageRoot('/tmp')).toThrowError()
  })
})

describe('getAllConfigFilesInDirectory', () => {
  it('returns default and local files', () => {
    const files = getAllConfigFilesInDirectory('/test', undefined)
    expect(files).toEqual([
      '/test/default.yaml',
      '/test/local.yaml',
    ])
  })
  it('returns default, environment, and local files', () => {
    const files = getAllConfigFilesInDirectory('/test', 'test')
    expect(files).toEqual([
      '/test/default.yaml',
      '/test/test.yaml',
      '/test/local.yaml',
    ])
  })
})

describe('getAllConfigLocations', () => {
  it('returns package and local dirs when separate', () => {
    const files = getAllConfigLocations('/test1', '/test2', 'test')
    expect(files).toEqual([
      '/test1/default.yaml',
      '/test1/test.yaml',
      '/test1/local.yaml',
      '/test2/default.yaml',
      '/test2/test.yaml',
      '/test2/local.yaml',
    ])
  })

  it('returns only package dir when local dir is same', () => {
    const files = getAllConfigLocations('/test', '/test', 'test')
    expect(files).toEqual([
      '/test/default.yaml',
      '/test/test.yaml',
      '/test/local.yaml',
    ])
  })
  it('adds an extra config file', () => {
    const files = getAllConfigLocations(
      '/test', '/test', 'test', '/test/test-extra.yaml')
    expect(files).toEqual([
      '/test/default.yaml',
      '/test/test.yaml',
      '/test/local.yaml',
      '/test/test-extra.yaml',
    ])
  })
  it('does not add extra config file when it is the same', () => {
    const files = getAllConfigLocations(
      '/test', '/test', 'test', '/test/test.yaml')
    expect(files).toEqual([
      '/test/default.yaml',
      '/test/test.yaml',
      '/test/local.yaml',
    ])
  })
})

describe('toSnakeCase', () => {
  it('converts uppercase, underscore-separated words to snake case', () => {
    expect(toSnakeCase('TEST')).toBe('test')
    expect(toSnakeCase('TEST_1')).toBe('test1')
    expect(toSnakeCase('TEST_VALUE')).toBe('testValue')
    expect(toSnakeCase('TEST_VALUE_TWO')).toBe('testValueTwo')
  })
})

describe('mergeConfig', () => {
  it('merges source config into destination', () => {
    const dest = {
      a: 1,
      b: [2],
      c: {
        d: 3,
      },
    }
    expect(mergeConfig({
      a: 4,
      b: {value: 5},
      c: {
        e: 6,
      },
    }, dest)).toEqual({
      a: 4,
      b: {value: 5},
      c: {
        d: 3,
        e: 6,
      },
    })
  })
})

describe('readConfig', () => {

  it('reads from a number of files', () => {
    const result = readConfig()
    expect(result).toBeInstanceOf(ReadConfig)
  })

  it('reads from an extra config file', () => {
  })

  describe('errors', () => {
    const dir = join(__dirname, 'test.dir')
    beforeEach(() => {
      mkdirSync(dir)
    })
    afterEach(() => {
      rmdirSync(dir)
    })
    it('fails on errors different than ENOENT', () => {
      expect(() => readConfig(
        process.env,
        undefined,
        undefined,
        dir,
      )).toThrowError(/EISDIR/)
    })
  })

  it('does not fail when no config files found', () => {
    readConfig({}, '/tmp', '/tmp')
  })

  it('reads values from environment variables', () => {
    const config = readConfig({
      PEERCALLS__TEST_VALUE__SUB_VALUE_1: '1',
      PEERCALLS__TEST_VALUE__SUB_VALUE_2: JSON.stringify({a: 2}),
      PEERCALLS__TEST_VALUE__SUB_VALUE_3: 'string',
    }, '/tmp', '/tmp')
    expect(config.value()).toEqual({
      testValue: {
        subValue1: 1,
        subValue2: {a: 2},
        subValue3: 'string',
      },
    })
  })

})
