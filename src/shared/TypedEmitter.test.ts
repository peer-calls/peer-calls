import { TypedEmitter } from './TypedEmitter'
import { EventEmitter } from 'events'

describe('TypedEmitter', () => {

  let stringMock: jest.Mock<void, [string]>
  let numberMock: jest.Mock<void, [number]>
  let valueMock: jest.Mock<void, [Value]>

  beforeEach(() => {
    stringMock = jest.fn()
    numberMock = jest.fn()
    valueMock = jest.fn()
  })

  const listener1 = (arg: string) => {
    stringMock(arg)
  }

  const listener2 = (arg: number) => {
    numberMock(arg)
  }

  const listener3 = (arg: Value) => {
    valueMock(arg)
  }

  interface Value {
    a: number
  }

  interface Events {
    test1: string
    test2: number
    test3: Value
  }

  let emitter: TypedEmitter<Events>
  beforeEach(() => {
    emitter = new EventEmitter()
    emitter.on('test1', listener1)
    emitter.on('test2', listener2)
    emitter.once('test3', listener3)
  })

  describe('on & on', () => {
    it('adds an event emitter', () => {
      emitter.emit('test1', 'value')
      emitter.emit('test2', 3)
      expect(stringMock.mock.calls).toEqual([[ 'value' ]])
      expect(numberMock.mock.calls).toEqual([[ 3 ]])
    })
  })

  describe('once', () => {
    it('adds an event emitter for one use only', () => {
      emitter.emit('test3', { a: 1 })
      emitter.emit('test3', { a: 2 })
      expect(valueMock.mock.calls).toEqual([[ { a: 1 } ]])
    })
  })

  describe('removeListener', () => {
    it('removes an event listener', () => {
      emitter.removeListener('test1', listener1)
      emitter.removeListener('test2', listener2)
      emitter.emit('test1', 'value')
      emitter.emit('test2', 3)
      expect(stringMock.mock.calls).toEqual([])
      expect(numberMock.mock.calls).toEqual([])
    })
  })

})
