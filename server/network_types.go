package server

import (
	"github.com/juju/errors"
	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/pion/webrtc/v3"
)

func NewNetworkTypes(log logger.Logger, networkTypes []string) (ret []webrtc.NetworkType) {
	for _, networkType := range networkTypes {
		nt, err := webrtc.NewNetworkType(networkType)
		if err != nil {
			log.Error(errors.Annotatef(err, "Invalid network type: %s", networkType), nil)

			continue
		}

		ret = append(ret, nt)
	}

	return ret
}
