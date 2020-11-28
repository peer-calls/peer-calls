package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"
)

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
	case AuthTypeNone:
		fallthrough
	default:
		return ICEAuthServer{
			URLs:       server.URLs,
			Credential: "",
			Username:   "",
		}
	}
}

const oneHourSeconds = 24 * 3600

func getICEStaticAuthSecretCredentials(server ICEServer) ICEAuthServer {
	timestamp := time.Now().Unix() + oneHourSeconds
	username := fmt.Sprintf("%d:%s", timestamp, server.AuthSecret.Username)
	h := hmac.New(sha1.New, []byte(server.AuthSecret.Secret))

	if _, err := h.Write([]byte(username)); err != nil {
		// Should never happen.
		panic(fmt.Sprintf("write to hmac failed: %+v", err))
	}

	credential := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return ICEAuthServer{
		URLs:       server.URLs,
		Username:   username,
		Credential: credential,
	}
}
