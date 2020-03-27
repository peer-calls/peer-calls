package logger_test

import (
	"os"
	"strings"
	"testing"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/stretchr/testify/assert"
)

func TestGetLogger_Printf(t *testing.T) {
	defer os.Unsetenv("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "a,b")
	var out strings.Builder
	loggerFactory := logger.NewLoggerFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logC := loggerFactory.GetLogger("c")
	logB.Printf("Test B: %s", "test b")
	logC.Printf("Test C: %s", "test c")
	assert.Regexp(t, " \\[b] Test B: test b\n", out.String())
}

func TestGetLogger_Println(t *testing.T) {
	defer os.Unsetenv("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "a,b")
	var out strings.Builder
	loggerFactory := logger.NewLoggerFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logC := loggerFactory.GetLogger("c")
	logB.Println(1, "one")
	logC.Println(2, "two")
	assert.Regexp(t, " \\[b] 1 one\n", out.String())
}

func TestGetLogger_default(t *testing.T) {
	os.Unsetenv("TESTLOG_")
	var out strings.Builder
	loggerFactory := logger.NewLoggerFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logB.Println(1, "one")
	assert.Regexp(t, " \\[b] 1 one\n", out.String())
}
