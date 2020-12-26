package logger

import (
	"strings"
)

// Config describes an interface which provides a method for getting a logging
// level for a particular namespace.
type Config interface {
	// LevelForNamespace returns a logging Level for particular namespace.
	LevelForNamespace(namespace string) Level
}

// ConfigMap reads the configuration from a CSV string. For example it can
// be easily used for reading the configuration from an environment variable.
type ConfigMap map[string]Level

// NewStringConfig parses the provided string and returns Config.
func NewConfigMapFromString(stringConfig string) Config {
	if stringConfig == "" {
		return ConfigMap{}
	}

	configSlice := strings.Split(stringConfig, ",")

	ret := make(ConfigMap, len(configSlice))

	for _, ns := range strings.Split(stringConfig, ",") {
		level := LevelInfo

		if index := strings.LastIndex(ns, ":"); index > -1 {
			if cfgLevel, ok := LevelFromString(ns[index+1:]); ok {
				level = cfgLevel
				ns = ns[:index]
			}
		}

		ret[ns] = level
	}

	return ret
}

// LevelForNamespace implements Config.
func (c ConfigMap) LevelForNamespace(namespace string) Level {
	level, ok := c[namespace]
	if ok {
		return level
	}

	// Return configuration for root logger.
	return c["*"]
}
