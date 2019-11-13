import * as turn from './turn'
import { ICEServer } from './config'

describe('server/turn', () => {
  describe('getCredentials', () => {
    it('returns username & credential', () => {
      const auth = turn.getCredentials('a', 'b')
      expect(auth).toEqual(jasmine.any(Object))
      expect(auth.username).toEqual(jasmine.any(String))
      expect(auth.credential).toEqual(jasmine.any(String))
    })
  })

  describe('processServers', () => {
    const servers: ICEServer[] = [{
      url: 'server1',
      urls: 'server1',
      auth: undefined,
      username: 'a',
      credential: 'b',
    }, {
      url: 'server2',
      urls: 'server2',
      username: 'c',
      secret: 'd',
      auth: 'secret',
    }]

    it('does not expose secret', () => {
      const s = turn.processServers(servers)
      expect(s.length).toBe(2)
      expect(s[0]).toBe(servers[0])
      expect(s[1]).toEqual({
        url: 'server2',
        urls: 'server2',
        username: jasmine.any(String),
        credential: jasmine.any(String),
      })
      expect(s[1].username).toMatch(/^[0-9]+:c$/)
    })

    it('throws error when unknown auth type', () => {
      expect(() => turn.processServers([{ auth: 'bla' } as any]))
      .toThrowError(/not implemented/)
    })
  })
})
