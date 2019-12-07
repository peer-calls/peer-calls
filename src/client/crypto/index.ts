export async function generateECDHKeyPair() {
  const key = await window.crypto.subtle.generateKey(
    {
      name: 'ECDH',
      namedCurve: 'P-256',
    },
    /* extractable */ false,
    ['deriveKey'],
  )
  return key
}

export async function deriveECDHKey(params: {
  privateKey: CryptoKey
  publicKey: CryptoKey
}) {
  const { privateKey, publicKey } = params
  const derivedKey = await window.crypto.subtle.deriveKey(
    {
      name: 'ECDH',
      public: publicKey,
    },
    privateKey,
    {
      name: 'AES-CTR',
      length: 256,
    },
    /* extractable */ false,
    ['encrypt', 'decrypt'],
  )

  return derivedKey
}

export async function encrypt(key: CryptoKey, data: string): Promise<string> {
  const encrypted = await window.crypto.subtle.encrypt(
    {
      name: 'AES-CTR',
      counter: new Uint8Array(16),
      length: 128,
    },
    key,
    str2ab(data),
  )
  return ab2str(encrypted)
}

export async function decrypt(key: CryptoKey, data: string): Promise<string> {
  const arrayBuffer = str2ab(data)
  const decrypted = await window.crypto.subtle.decrypt(
    {
      name: 'AES-CTR',
      counter: new ArrayBuffer(16),
      length: 128,
    },
    key,
    arrayBuffer,
  )
  return ab2str(decrypted)
}


function ab2str(buf: ArrayBuffer): string {
  return String.fromCharCode.apply(
    null, new Uint16Array(buf) as unknown as number[])
}

function str2ab(str: string): ArrayBuffer {
  const buf = new ArrayBuffer(str.length*2) // 2 bytes for each char
  const bufView = new Uint16Array(buf)
  for (let i=0, strLen=str.length; i < strLen; i++) {
    bufView[i] = str.charCodeAt(i)
  }
  return buf
}
