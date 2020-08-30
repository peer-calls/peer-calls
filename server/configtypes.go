package server

type AuthType string

const (
	AuthTypeSecret AuthType = "secret"
	AuthTypeNone   AuthType = ""
)

type ICEServer struct {
	URLs       []string `yaml:"urls"`
	AuthType   AuthType `yaml:"auth_type"`
	AuthSecret struct {
		Username string `yaml:"username"`
		Secret   string `yaml:"secret"`
	} `yaml:"auth_secret"`
}

type TLSConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type StoreType string

const (
	StoreTypeMemory StoreType = "memory"
	StoreTypeRedis  StoreType = "redis"
)

type RedisConfig struct {
	Host   string `yaml:"host"`
	Port   int    `yaml:"port"`
	Prefix string `yaml:"prefix"`
}

type StoreConfig struct {
	Type  StoreType   `yaml:"type"`
	Redis RedisConfig `yaml:"redis"`
}

type NetworkType string

const (
	NetworkTypeMesh NetworkType = "mesh"
	NetworkTypeSFU  NetworkType = "sfu"
)

type NetworkConfig struct {
	Type NetworkType      `yaml:"type"`
	SFU  NetworkConfigSFU `yaml:"sfu"`
}

type NetworkConfigSFU struct {
	Interfaces    []string `yaml:"interfaces"`
	JitterBuffer  bool     `yaml:"jitter_buffer"`
	Protocols     []string `yaml:"protocols"`
	TCPBindAddr   string   `yaml:"tcp_bind_addr"`
	TCPListenPort int      `yaml:"tcp_listen_port"`
	UDP           struct {
		PortMin uint16 `yaml:"port_min"`
		PortMax uint16 `yaml:"port_max"`
	} `yaml:"udp"`
}

type PrometheusConfig struct {
	AccessToken string `yaml:"access_token"`
}

type Config struct {
	BaseURL    string           `yaml:"base_url"`
	BindHost   string           `yaml:"bind_host"`
	BindPort   int              `yaml:"bind_port"`
	ICEServers []ICEServer      `yaml:"ice_servers"`
	TLS        TLSConfig        `yaml:"tls"`
	Store      StoreConfig      `yaml:"store"`
	Network    NetworkConfig    `yaml:"network"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

type ICEAuthServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

type ClientConfig struct {
	BaseURL    string          `json:"baseUrl"`
	Nickname   string          `json:"nickname"`
	CallID     string          `json:"callId"`
	UserID     string          `json:"userId"`
	ICEServers []ICEAuthServer `json:"iceServers"`
	Network    NetworkType     `json:"network"`
}
