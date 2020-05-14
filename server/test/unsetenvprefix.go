package test

import (
	"os"
	"strings"
)

func UnsetEnvPrefix(prefix string) {
	for _, v := range os.Environ() {
		if strings.HasPrefix(v, prefix) {
			os.Unsetenv(strings.Split(v, "=")[0])
		}
	}
}
