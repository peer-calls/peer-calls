package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"
)

type ICEAuthServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

func GetICEAuthServers(servers []ICEServer) (result []ICEAuthServer) {
	for _, server := range servers {
		result = append(result, newICEServer(server))
	}
	return
}

func newICEServer(server ICEServer) ICEAuthServer {
	switch server.AuthType {
	case AuthTypeSecret:
		return getICEStaticAuthSecretCredentials(server)
	default:
		return ICEAuthServer{URLs: server.URLs}
	}
}

func getICEStaticAuthSecretCredentials(server ICEServer) ICEAuthServer {
	timestamp := time.Now().Unix() + 24*3600
	username := fmt.Sprintf("%d:%s", timestamp, server.AuthSecret.Username)
	h := hmac.New(sha1.New, []byte(server.AuthSecret.Secret))
	h.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return ICEAuthServer{
		URLs:       server.URLs,
		Username:   username,
		Credential: credential,
	}
}
