let success = false

export const insertableStreamsCodec = {
  mockSuccess(ok: boolean) {
    success = ok
  },

  setPassword: jest.fn().mockImplementation(() => {
    return success
  }),

  encrypt: jest.fn().mockImplementation(() => {
    return success
  }),

  decrypt: jest.fn().mockImplementation(() => {
    return success
  }),
}
