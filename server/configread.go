package server

import (
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/juju/errors"
	"gopkg.in/yaml.v2"
)

func ReadConfigFile(filename string, c *Config) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Annotatef(err, "read config file: %s", filename)
	}

	defer f.Close()

	err = ReadConfigYAML(f, c)

	return errors.Annotatef(err, "read yaml config: %s", filename)
}

func ReadConfigFiles(filenames []string, c *Config) (err error) {
	for _, filename := range filenames {
		err = ReadConfigFile(filename, c)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func InitConfig(c *Config) {
	c.BindPort = 3000
	c.Network.Type = NetworkTypeMesh
	c.Store.Type = StoreTypeMemory
	c.ICEServers = []ICEServer{{
		URLs: []string{"stun:stun.l.google.com:19302"},
	}, {
		URLs: []string{"stun:global.stun.twilio.com:3478?transport=udp"},
	}}
}

func ReadConfig(filenames []string) (c Config, err error) {
	InitConfig(&c)
	err = ReadConfigFiles(filenames, &c)
	ReadConfigFromEnv("PEERCALLS_", &c)

	return c, errors.Trace(err)
}

func ReadConfigYAML(reader io.Reader, c *Config) error {
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(c); err != nil {
		return errors.Annotatef(err, "decode yaml")
	}

	return nil
}

func ReadConfigFromEnv(prefix string, c *Config) {
	setEnvString(&c.BaseURL, prefix+"BASE_URL")
	setEnvString(&c.BindHost, prefix+"BIND_HOST")
	setEnvInt(&c.BindPort, prefix+"BIND_PORT")
	setEnvString(&c.TLS.Cert, prefix+"TLS_CERT")
	setEnvString(&c.TLS.Key, prefix+"TLS_KEY")

	setEnvString(&c.FS, prefix+"FS")

	setEnvStoreType(&c.Store.Type, prefix+"STORE_TYPE")
	setEnvString(&c.Store.Redis.Host, prefix+"STORE_REDIS_HOST")
	setEnvInt(&c.Store.Redis.Port, prefix+"STORE_REDIS_PORT")
	setEnvString(&c.Store.Redis.Prefix, prefix+"STORE_REDIS_PREFIX")

	setEnvNetworkType(&c.Network.Type, prefix+"NETWORK_TYPE")
	setEnvString(&c.Network.SFU.TCPBindAddr, prefix+"NETWORK_SFU_TCP_BIND_ADDR")
	setEnvInt(&c.Network.SFU.TCPListenPort, prefix+"NETWORK_SFU_TCP_LISTEN_PORT")
	setEnvStringArray(&c.Network.SFU.Protocols, prefix+"NETWORK_SFU_PROTOCOLS")
	setEnvStringArray(&c.Network.SFU.Interfaces, prefix+"NETWORK_SFU_INTERFACES")
	setEnvBool(&c.Network.SFU.JitterBuffer, prefix+"NETWORK_SFU_JITTER_BUFFER")
	setEnvStringArray(&c.Network.SFU.Transport.Nodes, prefix+"NETWORK_SFU_TRANSPORT_NODES")
	setEnvString(&c.Network.SFU.Transport.ListenAddr, prefix+"NETWORK_SFU_TRANSPORT_LISTEN_ADDR")
	setEnvUint16(&c.Network.SFU.UDP.PortMin, prefix+"NETWORK_SFU_UDP_PORT_MIN")
	setEnvUint16(&c.Network.SFU.UDP.PortMax, prefix+"NETWORK_SFU_UDP_PORT_MAX")

	if value, ok := os.LookupEnv(prefix + "ICE_SERVER_URLS"); ok {
		// Do not use the default servers, even if value is empty.
		c.ICEServers = make([]ICEServer, 0, 1)
		// Do not use the d
		var ice ICEServer

		setSlice(&ice.URLs, value)

		if len(ice.URLs) > 0 {
			setEnvAuthType(&ice.AuthType, prefix+"ICE_SERVER_AUTH_TYPE")
			setEnvString(&ice.AuthSecret.Secret, prefix+"ICE_SERVER_SECRET")
			setEnvString(&ice.AuthSecret.Username, prefix+"ICE_SERVER_USERNAME")
			c.ICEServers = append(c.ICEServers, ice)
		}
	}

	setEnvString(&c.Prometheus.AccessToken, prefix+"PROMETHEUS_ACCESS_TOKEN")

	setEnvBool(&c.Frontend.EncodedInsertableStreams, prefix+"FRONTEND_ENCODED_INSERTABLE_STREAMS")
}

func setEnvSlice(dest *[]string, name string) {
	value := os.Getenv(name)
	setSlice(dest, value)
}

func setSlice(dest *[]string, value string) {
	for _, v := range strings.Split(value, ",") {
		if v != "" {
			*dest = append(*dest, v)
		}
	}
}

func setEnvString(dest *string, name string) {
	value := os.Getenv(name)
	if value != "" {
		*dest = value
	}
}

func setEnvInt(dest *int, name string) {
	value, err := strconv.Atoi(os.Getenv(name))
	if err == nil {
		*dest = value
	}
}

func setEnvUint16(dest *uint16, name string) {
	value, err := strconv.ParseUint(os.Getenv(name), 10, 32)
	if err == nil {
		*dest = uint16(value)
	}
}

func setEnvBool(dest *bool, name string) {
	val := os.Getenv(name)

	// Only set the boolean value when the environment variable is explicitly set
	// to either 'true' or 'false', to prevent resetting the pointer value to
	// false when there is no environment variable defined.
	switch val {
	case "true":
		*dest = true
	case "false":
		*dest = false
	}
}

func setEnvAuthType(authType *AuthType, name string) {
	value := os.Getenv(name)
	switch AuthType(value) {
	case AuthTypeSecret:
		*authType = AuthTypeSecret
	case AuthTypeNone:
		*authType = AuthTypeNone
	}
}

func setEnvNetworkType(networkType *NetworkType, name string) {
	value := os.Getenv(name)
	switch NetworkType(value) {
	case NetworkTypeMesh:
		*networkType = NetworkTypeMesh
	case NetworkTypeSFU:
		*networkType = NetworkTypeSFU
	}
}

func setEnvStoreType(storeType *StoreType, name string) {
	value := os.Getenv(name)
	switch StoreType(value) {
	case StoreTypeRedis:
		*storeType = StoreTypeRedis
	case StoreTypeMemory:
		*storeType = StoreTypeMemory
	}
}

func setEnvStringArray(interfaces *[]string, name string) {
	value := os.Getenv(name)
	if value != "" {
		values := strings.Split(value, ",")
		*interfaces = values
	}
}
