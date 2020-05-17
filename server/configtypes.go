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
	Interfaces   []string `yaml:"interfaces"`
	JitterBuffer bool     `yaml:"jitter_buffer"`
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
