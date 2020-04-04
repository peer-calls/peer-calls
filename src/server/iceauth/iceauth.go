package iceauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jeremija/peer-calls/src/server/config"
)

type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

func GetICEServers(servers []config.ICEServer) (result []ICEServer) {
	for _, server := range servers {
		result = append(result, getICEServer(server))
	}
	return
}

func getICEServer(server config.ICEServer) ICEServer {
	switch server.AuthType {
	case config.AuthTypeSecret:
		return getSecretCredentials(server)
	default:
		return ICEServer{URLs: server.URLs}
	}
}

func getSecretCredentials(server config.ICEServer) ICEServer {
	timestamp := time.Now().UnixNano() / 1_000_000
	username := fmt.Sprintf("%d:%s", timestamp, server.AuthSecret.Username)
	h := hmac.New(sha1.New, []byte(server.AuthSecret.Secret))
	h.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return ICEServer{
		URLs:       server.URLs,
		Username:   username,
		Credential: credential,
	}
}
