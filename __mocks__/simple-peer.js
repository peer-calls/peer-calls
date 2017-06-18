import EventEmitter from 'events'
const Peer = jest.genMockFunction().mockImplementation(() => {
  let peer = new EventEmitter()
  peer.destroy = jest.fn()
  peer.signal = jest.fn()
  peer.send = jest.fn()
  Peer.instances.push(peer)
  return peer
})
Peer.instances = []
export default Peer
