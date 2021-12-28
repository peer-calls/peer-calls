package server_test

import (
	"testing"

	"github.com/peer-calls/peer-calls/v4/server"
	"github.com/peer-calls/peer-calls/v4/server/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestRoomManagerFactory(t *testing.T) {
	goleak.VerifyNone(t)
	defer goleak.VerifyNone(t)

	log := test.NewLogger()

	tracksManager := newMockTracksManager()
	adapterFactory := server.NewAdapterFactory(log, server.StoreConfig{
		Type: server.StoreTypeMemory,
	})

	defer adapterFactory.Close()

	factory := server.NewRoomManagerFactory(server.RoomManagerFactoryParams{
		Log:            log,
		AdapterFactory: adapterFactory,
		TracksManager:  tracksManager,
	})

	cleanup := func(rm server.RoomManager, nm *server.NodeManager) {
		if crm, ok := rm.(*server.ChannelRoomManager); ok {
			crm.Close()
		}

		if nm != nil {
			nm.Close()
		}
	}

	t.Run("mesh", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeMesh

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.AdapterRoomManager)
		assert.True(t, ok)
		assert.Nil(t, nm)
	})

	t.Run("sfu default", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeSFU

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.AdapterRoomManager)
		assert.True(t, ok)
		assert.Nil(t, nm)
	})

	t.Run("sfu transport listen fallback", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeSFU
		networkConfig.SFU.Transport.ListenAddr = "invalid-addr:9999"

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.AdapterRoomManager)
		assert.True(t, ok, "should fall back to default")
		assert.Nil(t, nm, "should fall back to default")
	})

	t.Run("sfu transport listen ok", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeSFU
		networkConfig.SFU.Transport.ListenAddr = "127.0.0.1:0"

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.ChannelRoomManager)
		assert.True(t, ok)
		assert.NotNil(t, nm)
	})

	t.Run("sfu transport listen ok, remote addrs", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeSFU
		networkConfig.SFU.Transport.ListenAddr = "127.0.0.1:0"
		networkConfig.SFU.Transport.Nodes = []string{
			"127.0.0.1:1234",
			"127.0.0.1:1235",
		}

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.ChannelRoomManager)
		assert.True(t, ok)
		assert.NotNil(t, nm)
	})

	t.Run("sfu transport listen ok, invalid addrs", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeSFU
		networkConfig.SFU.Transport.ListenAddr = "127.0.0.1:0"
		networkConfig.SFU.Transport.Nodes = []string{
			"127.0.0.1:1234",
			"invalid-addr:1235",
		}

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.AdapterRoomManager)
		assert.True(t, ok, "should fall back to default")
		assert.Nil(t, nm, "should fall back to default")
	})

	t.Run("sfu transport listen ok, invalid addrs 2", func(t *testing.T) {
		networkConfig := server.NetworkConfig{}
		networkConfig.Type = server.NetworkTypeSFU
		networkConfig.SFU.Transport.ListenAddr = "127.0.0.1:0"
		networkConfig.SFU.Transport.Nodes = []string{
			"127.0.0.1:1234",
			"invalid-addr",
		}

		rm, nm := factory.NewRoomManager(networkConfig)
		defer cleanup(rm, nm)
		_, ok := rm.(*server.AdapterRoomManager)
		assert.True(t, ok, "should fall back to default")
		assert.Nil(t, nm, "should fall back to default")
	})
}
