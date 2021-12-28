package server

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

type RoomManagerFactory struct {
	params *RoomManagerFactoryParams
}

type RoomManagerFactoryParams struct {
	AdapterFactory *AdapterFactory
	TracksManager  TracksManager
	Log            logger.Logger
}

func NewRoomManagerFactory(params RoomManagerFactoryParams) *RoomManagerFactory {
	params.Log = params.Log.WithNamespaceAppended("room_manager_factory")

	return &RoomManagerFactory{
		params: &params,
	}
}

func (rmf *RoomManagerFactory) NewRoomManager(c NetworkConfig) (RoomManager, *NodeManager) {
	rooms := NewAdapterRoomManager(rmf.params.AdapterFactory.NewAdapter)

	if c.Type == NetworkTypeSFU && c.SFU.Transport.ListenAddr != "" {
		roomManager, nodeManager, err := rmf.createChannelRoomManager(c, rooms)
		if err == nil {
			return roomManager, nodeManager
		}

		rmf.params.Log.Info("Error creating NodeTransport, falling back to single SFU", nil)
	}

	return rooms, nil
}

func (rmf *RoomManagerFactory) createChannelRoomManager(
	c NetworkConfig,
	rooms RoomManager,
) (*ChannelRoomManager, *NodeManager, error) {
	listenAddr, err := ParseUDPAddr(c.SFU.Transport.ListenAddr)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "parse UDP addr")
	}

	nodes, err := ParseUDPAddrs(c.SFU.Transport.Nodes)
	if err != nil {
		return nil, nil, errors.Annotatef(err, "parse UDP addrs")
	}

	channelRoomManager := NewChannelRoomManager(rooms)

	nodeManager, err := NewNodeManager(NodeManagerParams{
		Log:           rmf.params.Log,
		ListenAddr:    listenAddr,
		Nodes:         nodes,
		RoomManager:   channelRoomManager,
		TracksManager: rmf.params.TracksManager,
	})
	if err != nil {
		channelRoomManager.Close()

		return nil, nil, errors.Annotatef(err, "new node manager")
	}

	return channelRoomManager, nodeManager, nil
}
