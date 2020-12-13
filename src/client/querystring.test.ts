import { parseQueryString } from './querystring'

describe('querystring', () => {

  describe('parseQueryString', () => {
    it('parses simple query strings', () => {
      expect(parseQueryString('?a=b&c=1')).toEqual({
        a: 'b',
        c: '1',
      })
    })
    it('parses encoded query strings', () => {
      const qs = '?' +
        encodeURIComponent('a=b') + '=' +
        encodeURIComponent('c=d') + '&' +
        encodeURIComponent('e f') + '=' +
        encodeURIComponent('g h')
      expect(qs).toEqual('?a%3Db=c%3Dd&e%20f=g%20h')
      expect(parseQueryString(qs)).toEqual({
        'a=b': 'c=d',
        'e f': 'g h',
      })
    })
  })

})
