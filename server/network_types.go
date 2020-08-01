package server

import "github.com/pion/webrtc/v3"

func NewNetworkTypes(logger Logger, networkTypes []string) (ret []webrtc.NetworkType) {
	for _, networkType := range networkTypes {
		nt, err := webrtc.NewNetworkType(networkType)

		if err != nil {
			logger.Printf("Invalid network type: %s", networkType)
			continue
		}

		ret = append(ret, nt)
	}

	return ret
}
