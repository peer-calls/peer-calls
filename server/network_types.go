package server

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/pion/webrtc/v3"
)

func NewNetworkTypes(log logger.Logger, networkTypes []string) (ret []webrtc.NetworkType) {
	log = log.WithNamespaceAppended("network_types")

	for _, networkType := range networkTypes {
		nt, err := webrtc.NewNetworkType(networkType)
		if err != nil {
			log.Error("NewNetworkType", errors.Trace(err), nil)

			continue
		}

		ret = append(ret, nt)
	}

	return ret
}
