package logger_test

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/peer-calls/peer-calls/v4/server/logger"
	"github.com/stretchr/testify/assert"
)

type testWriter struct {
	mockErr error
	b       strings.Builder
}

func newTestWriter() *testWriter {
	return &testWriter{}
}

func (t *testWriter) Write(b []byte) (int, error) {
	if t.mockErr != nil {
		return 0, t.mockErr
	}

	return t.b.Write(b)
}

func (t *testWriter) String() string {
	return t.b.String()
}

type testFormatter struct {
	*logger.StringFormatter
	mockErr error
}

func newTestFormatter() *testFormatter {
	return &testFormatter{
		StringFormatter: logger.NewStringFormatter(logger.StringFormatterParams{
			DateLayout:               "-",
			DisableContextKeySorting: false,
		}),
	}
}

func (f *testFormatter) Format(message logger.Message) ([]byte, error) {
	if f.mockErr != nil {
		return nil, f.mockErr
	}

	return f.StringFormatter.Format(message)
}

var errTest = fmt.Errorf("test err")

func TestLogger_Namespace(t *testing.T) {
	t.Parallel()

	log := logger.New().WithNamespace("test").WithNamespaceAppended("test2")

	assert.Equal(t, "test:test2", log.Namespace())
}

func TestLogger(t *testing.T) {
	t.Parallel()

	type testEntry struct {
		namespace string
		level     logger.Level
		message   string
		err       error
		ctx       logger.Ctx
	}

	type testCase struct {
		config           string
		ctx              logger.Ctx
		entries          []testEntry
		mockWriterErr    error
		mockFormatterErr error
		wantErr          error
		wantResult       string
	}

	testCases := []testCase{
		{
			config: "",
			entries: []testEntry{
				{"a", logger.LevelInfo, "test", nil, nil},
			},
			wantResult: "",
		},
		{
			config: "a:b",
			entries: []testEntry{
				{"a:b", logger.LevelInfo, "test", nil, nil},
			},
			wantResult: "-  info [                 a:b] test\n",
		},
		{
			config: "a",
			entries: []testEntry{
				{"a", logger.LevelDebug, "test", nil, nil},
			},
		},
		{
			config: "a",
			entries: []testEntry{
				{"b", logger.LevelInfo, "test", nil, nil},
			},
		},
		{
			config: "a:b:debug",
			entries: []testEntry{
				{"a:b", logger.LevelDebug, "test", nil, nil},
			},
			wantResult: "- debug [                 a:b] test\n",
		},
		{
			config: "a:b:debug",
			entries: []testEntry{
				{"a:b", logger.LevelDebug, "test", nil, nil},
			},
			mockWriterErr: errTest,
			wantErr:       fmt.Errorf("log write error: %w", errTest),
		},
		{
			config: "a:b:debug",
			entries: []testEntry{
				{"a:b", logger.LevelDebug, "test", nil, nil},
			},
			mockFormatterErr: errTest,
			wantErr:          fmt.Errorf("log format error: %w", errTest),
		},
		{
			config: "a:b:debug",
			ctx: logger.Ctx{
				"k1": "v1",
				"k2": "v2",
			},
			entries: []testEntry{
				{"a:b", logger.LevelDebug, "test", nil, logger.Ctx{"k2": "v3"}},
			},
			wantResult: "- debug [                 a:b] test k1=v1 k2=v3\n",
		},
		{
			config: "*:b:trace",
			entries: []testEntry{
				{"a:b", logger.LevelTrace, "test1", nil, nil},
				{"a:c", logger.LevelTrace, "test2", nil, nil},
			},
			wantResult: "- trace [                 a:b] test1\n",
		},
		{
			config: ":debug",
			entries: []testEntry{
				{"a:b", logger.LevelDebug, "test1", nil, nil},
				{"a:c", logger.LevelTrace, "test2", nil, nil},
			},
			wantResult: "- debug [                 a:b] test1\n",
		},
		{
			config: "a:b:trace,c:d",
			ctx: logger.Ctx{
				"k1": "v1",
				"k2": "v2",
			},
			entries: []testEntry{
				{"a:b", logger.LevelTrace, "test1", nil, logger.Ctx{"k2": "v3"}},
				{"a:b", logger.LevelDebug, "test2", nil, logger.Ctx{"k3": "v3"}},
				{"a:b", logger.LevelInfo, "test3", nil, logger.Ctx{"k4": "v4"}},
				{"a:b", logger.LevelWarn, "test4", nil, logger.Ctx{"k5": "v5"}},
				{"a:b", logger.LevelError, "", errTest, logger.Ctx{"k6": "v6"}},
				{"a:b", logger.LevelError, "err msg", errTest, logger.Ctx{"k7": "v7"}},
				{"a:b", logger.LevelError, "err msg", nil, logger.Ctx{"k8": "v8"}},
				{"c:d", logger.LevelTrace, "test1", nil, logger.Ctx{"k2": "v3"}},
				{"c:d", logger.LevelDebug, "test2", nil, logger.Ctx{"k3": "v3"}},
				{"c:d", logger.LevelInfo, "test3", nil, logger.Ctx{"k4": "v4"}},
				{"c:d", logger.LevelWarn, "test4", nil, logger.Ctx{"k5": "v5"}},
				{"c:d", logger.LevelError, "", errTest, logger.Ctx{"k6": "v6"}},
				{"e:f", logger.LevelTrace, "test1", nil, logger.Ctx{"k2": "v3"}},
				{"e:f", logger.LevelDebug, "test2", nil, logger.Ctx{"k3": "v3"}},
				{"e:f", logger.LevelInfo, "test3", nil, logger.Ctx{"k4": "v4"}},
				{"e:f", logger.LevelWarn, "test4", nil, logger.Ctx{"k5": "v5"}},
				{"e:f", logger.LevelError, "", errTest, logger.Ctx{"k6": "v6"}},
			},
			wantResult: `- trace [                 a:b] test1 k1=v1 k2=v3
- debug [                 a:b] test2 k1=v1 k2=v2 k3=v3
-  info [                 a:b] test3 k1=v1 k2=v2 k4=v4
-  warn [                 a:b] test4 k1=v1 k2=v2 k5=v5
- error [                 a:b] test err k1=v1 k2=v2 k6=v6
- error [                 a:b] err msg: test err k1=v1 k2=v2 k7=v7
- error [                 a:b] err msg k1=v1 k2=v2 k8=v8
-  info [                 c:d] test3 k1=v1 k2=v2 k4=v4
-  warn [                 c:d] test4 k1=v1 k2=v2 k5=v5
- error [                 c:d] test err k1=v1 k2=v2 k6=v6
`,
		},
	}

	for i, tc := range testCases {
		descr := fmt.Sprintf("test case: %d", i)

		w := newTestWriter()

		formatter := newTestFormatter()

		root := logger.New().WithConfig(logger.NewConfigFromString(tc.config)).
			WithWriter(w).
			WithCtx(tc.ctx).
			WithFormatter(formatter)

		w.mockErr = tc.mockWriterErr
		formatter.mockErr = tc.mockFormatterErr

		var gotErr error

		for _, entry := range tc.entries {
			switch entry.level {
			case logger.LevelError:
				_, gotErr = root.WithNamespace(entry.namespace).Error(entry.message, entry.err, entry.ctx)
			case logger.LevelWarn:
				_, gotErr = root.WithNamespace(entry.namespace).Warn(entry.message, entry.ctx)
			case logger.LevelInfo:
				_, gotErr = root.WithNamespace(entry.namespace).Info(entry.message, entry.ctx)
			case logger.LevelDebug:
				_, gotErr = root.WithNamespace(entry.namespace).Debug(entry.message, entry.ctx)
			case logger.LevelTrace:
				_, gotErr = root.WithNamespace(entry.namespace).Trace(entry.message, entry.ctx)
			case logger.LevelDisabled:
				fallthrough
			case logger.LevelUnknown:
				fallthrough
			default:
				panic(fmt.Sprintf("unexpected level: %s", entry.level))
			}
		}

		assert.Equal(t, tc.wantErr, gotErr, "%s: wantErr", descr)

		gotResult := w.String()

		assert.Equal(t, tc.wantResult, gotResult, "\n", "%s: wantResult", descr)
	}
}

func TestLogger_WithNamespaceAppended(t *testing.T) {
	t.Parallel()

	w := newTestWriter()

	log := logger.New().WithConfig(logger.LevelInfo).
		WithNamespaceAppended("a").
		WithNamespaceAppended("b").
		WithWriter(w).
		WithFormatter(newTestFormatter())

	_, err := log.Info("test1", nil)
	assert.NoError(t, err)

	_, err = log.Trace("test2", nil)
	assert.NoError(t, err)

	gotStr := w.String()

	assert.Equal(t, "-  info [                 a:b] test1\n", gotStr)
}

func TestLogger_Ctx(t *testing.T) {
	t.Parallel()

	log := logger.New()

	assert.Equal(t, logger.Ctx(nil), log.Ctx())
	assert.Equal(t, logger.Ctx{"a": "b"}, log.WithCtx(logger.Ctx{"a": "b"}).Ctx())
}

func TestNewFromEnv(t *testing.T) {
	t.Parallel()

	envKey := "TEST_LOG"

	old := os.Getenv(envKey)

	defer os.Setenv(envKey, old)

	os.Setenv(envKey, "**:a:trace,:info")

	log := logger.NewFromEnv(envKey)

	assert.Equal(t, true, log.IsLevelEnabled(logger.LevelInfo))
	assert.Equal(t, false, log.IsLevelEnabled(logger.LevelDebug))

	assert.Equal(t, true, log.WithNamespace("a").IsLevelEnabled(logger.LevelInfo))
	assert.Equal(t, true, log.WithNamespace("a").IsLevelEnabled(logger.LevelDebug))

	assert.Equal(t, true, log.WithNamespace("c:b:a").IsLevelEnabled(logger.LevelInfo))
	assert.Equal(t, true, log.WithNamespace("c:b:a").IsLevelEnabled(logger.LevelDebug))

	assert.Equal(t, true, log.WithNamespace("b").IsLevelEnabled(logger.LevelInfo))
	assert.Equal(t, false, log.WithNamespace("b").IsLevelEnabled(logger.LevelDebug))
}

func BenchmarkLogger_Disabled(b *testing.B) {
	log := logger.New().WithNamespace("test").WithConfig(logger.LevelDisabled)

	var thread int64

	b.RunParallel(func(pb *testing.PB) {
		curThread := atomic.AddInt64(&thread, 1)

		var n int64

		for pb.Next() {
			curN := atomic.AddInt64(&n, 1)
			_, _ = log.Info("benchmark", logger.Ctx{"thread": curThread, "n": curN})
		}
	})
}

func BenchmarkLogger_Enabled(b *testing.B) {
	log := logger.New().WithNamespace("test").WithConfig(logger.LevelInfo)

	var thread int64

	b.RunParallel(func(pb *testing.PB) {
		curThread := atomic.AddInt64(&thread, 1)

		var n int64

		for pb.Next() {
			curN := atomic.AddInt64(&n, 1)
			_, _ = log.Info("benchmark", logger.Ctx{"thread": curThread, "n": curN})
		}
	})
}

func BenchmarkLogger_EnabledWithoutSorting(b *testing.B) {
	log := logger.New().WithNamespace("test").WithConfig(logger.LevelInfo).
		WithFormatter(logger.NewStringFormatter(logger.StringFormatterParams{
			DateLayout:               "",
			DisableContextKeySorting: true,
		}))

	var thread int64

	b.RunParallel(func(pb *testing.PB) {
		curThread := atomic.AddInt64(&thread, 1)

		var n int64

		for pb.Next() {
			curN := atomic.AddInt64(&n, 1)
			_, _ = log.Info("benchmark", logger.Ctx{"thread": curThread, "n": curN})
		}
	})
}
