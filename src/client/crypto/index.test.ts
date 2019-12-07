import { encrypt, decrypt, generateECDHKeyPair, deriveECDHKey } from './index'

describe('crypto', () => {

  let keypair1: CryptoKeyPair
  let keypair2: CryptoKeyPair

  let derived1: CryptoKey
  let derived2: CryptoKey
  beforeAll(async () => {
    keypair1 = await generateECDHKeyPair()
    keypair2 = await generateECDHKeyPair()

    derived1 = await deriveECDHKey({
      privateKey: keypair1.privateKey,
      publicKey: keypair2.publicKey,
    })
    derived2 = await deriveECDHKey({
      privateKey: keypair2.privateKey,
      publicKey: keypair1.publicKey,
    })
  })

  describe('encrypt and decrypt', () => {
    it('can be encrypted with one pair and decrypted with other', async () => {
      const message = 'test message'
      const encrypted = await encrypt(derived1, message)
      const decrypted = await decrypt(derived2, encrypted)
      expect(decrypted).toEqual(message)
    })
  })

})
