package logger_test

import (
	"os"
	"strings"
	"testing"

	"github.com/peer-calls/peer-calls/server/logger"
	"github.com/peer-calls/peer-calls/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLogger_Printf(t *testing.T) {
	defer test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "a,b")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logC := loggerFactory.GetLogger("c")
	logB.Printf("Test B: %s", "test b")
	logC.Printf("Test C: %s", "test c")
	assert.Regexp(t, " \\[              b] Test B: test b\n", out.String())
}

func TestGetLogger_Println(t *testing.T) {
	defer test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "a,b")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logC := loggerFactory.GetLogger("c")
	logB.Println(1, "one")
	logC.Println(2, "two")
	assert.Regexp(t, " \\[              b] 1 one\n", out.String())
}

func TestGetLogger_default(t *testing.T) {
	test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "b")
	defer test.UnsetEnvPrefix("TESTLOG_")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logB.Println(1, "one")
	assert.Regexp(t, " \\[              b] 1 one\n", out.String())
}

func TestSetDefaultEnabled(t *testing.T) {
	test.UnsetEnvPrefix("TESTLOG_")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	b := logB.(*logger.WriterLogger)
	b.Enabled = false
	loggerFactory.SetDefaultEnabled([]string{"b"})
	assert.True(t, b.Enabled)
	logB.Println(1, "one")
	assert.Regexp(t, " \\[              b] 1 one\n", out.String())
}

func TestGetLogger_Wildcard_All(t *testing.T) {
	defer test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "*")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logB := loggerFactory.GetLogger("b")
	logB.Println(1, "one")
	assert.Regexp(t, " \\[              b] 1 one\n", out.String())
}

func TestGetLogger_Wildcard_Middle(t *testing.T) {
	defer test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "a:*:warn")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logAOneWarn := loggerFactory.GetLogger("a:one:warn")
	logAOneInfo := loggerFactory.GetLogger("a:one:info")
	logATwoWarn := loggerFactory.GetLogger("a:two:warn")
	logATwoInfo := loggerFactory.GetLogger("a:two:info")
	logBOneWarn := loggerFactory.GetLogger("b:one:warn")

	logAOneWarn.Println("a one warn")
	logAOneInfo.Println("a one info")
	logATwoWarn.Println("a two warn")
	logATwoInfo.Println("a two info")
	logBOneWarn.Println("b one warn")

	str := out.String()
	require.Greater(t, len(str), 0)

	result := strings.Split(strings.Trim(str, "\n"), "\n")
	require.Equal(t, 2, len(result))
	assert.Regexp(t, " \\[     a:one:warn] a one warn", result[0])
	assert.Regexp(t, " \\[     a:two:warn] a two warn", result[1])
}

func TestGetLogger_Wildcard_End(t *testing.T) {
	defer test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "a:*")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logAOneWarn := loggerFactory.GetLogger("a:one:warn")
	logATwoInfo := loggerFactory.GetLogger("a:two:info")
	logBOneWarn := loggerFactory.GetLogger("b:one:warn")

	logAOneWarn.Println("a one warn")
	logATwoInfo.Println("a two info")
	logBOneWarn.Println("b one warn")

	str := out.String()
	require.Greater(t, len(str), 0)

	result := strings.Split(strings.Trim(str, "\n"), "\n")
	require.Equal(t, 2, len(result))
	assert.Regexp(t, " \\[     a:one:warn] a one warn", result[0])
	assert.Regexp(t, " \\[     a:two:info] a two info", result[1])
}

func TestGetLogger_Wildcard_Disabled(t *testing.T) {
	defer test.UnsetEnvPrefix("TESTLOG_")
	os.Setenv("TESTLOG_LOG", "-a:*:warn,-a:*:info,*")
	var out strings.Builder
	loggerFactory := logger.NewFactoryFromEnv("TESTLOG_", &out)
	logAOneWarn := loggerFactory.GetLogger("a:one:warn")
	logATwoInfo := loggerFactory.GetLogger("a:two:info")
	logBOneWarn := loggerFactory.GetLogger("b:one:warn")

	logAOneWarn.Println("a one warn")
	logATwoInfo.Println("a two info")
	logBOneWarn.Println("b one warn")

	str := out.String()
	require.Greater(t, len(str), 0)

	result := strings.Split(strings.Trim(str, "\n"), "\n")
	require.Equal(t, 1, len(result))
	assert.Regexp(t, " \\[     b:one:warn] b one warn", result[0])
}
