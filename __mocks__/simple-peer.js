import EventEmitter from 'events'
const Peer = jest.genMockFunction().mockImplementation(() => {
  let peer = new EventEmitter()
  peer.destroy = jest.genMockFunction()
  peer.signal = jest.genMockFunction()
  Peer.instances.push(peer)
  return peer
})
Peer.instances = []
export default Peer
