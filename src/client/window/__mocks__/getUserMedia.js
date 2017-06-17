import Promise from 'bluebird'

class MediaStream {}

let shouldFail
export const fail = _fail => shouldFail = !!_fail
export const stream = new MediaStream()
export default function getUserMedia () {
  return !shouldFail
  ? Promise.resolve(stream)
  : Promise.reject(new Error('test'))
}

