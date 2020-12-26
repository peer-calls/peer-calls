package logger2_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/peer-calls/peer-calls/server/logger2"
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
	*logger2.StringFormatter
	mockErr error
}

func newTestFormatter() *testFormatter {
	return &testFormatter{
		StringFormatter: logger2.NewStringFormatter(logger2.StringFormatterParams{
			DateLayout:               "-",
			DisableContextKeySorting: false,
		}),
	}
}

func (f *testFormatter) Format(ctx logger2.Ctx) ([]byte, error) {
	if f.mockErr != nil {
		return nil, f.mockErr
	}

	return f.StringFormatter.Format(ctx)
}

var errTest = fmt.Errorf("test err")

func TestLogger(t *testing.T) {
	t.Parallel()

	type testEntry struct {
		namespace string
		level     logger2.Level
		message   string
		err       error
		ctx       logger2.Ctx
	}

	type testCase struct {
		config           string
		ctx              logger2.Ctx
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
				{"a", logger2.LevelInfo, "test", nil, nil},
			},
			wantResult: "",
		},
		{
			config: "a:b",
			entries: []testEntry{
				{"a:b", logger2.LevelInfo, "test", nil, nil},
			},
			wantResult: "-  info [                 a:b] test\n",
		},
		{
			config: "a",
			entries: []testEntry{
				{"a", logger2.LevelDebug, "test", nil, nil},
			},
		},
		{
			config: "a",
			entries: []testEntry{
				{"b", logger2.LevelInfo, "test", nil, nil},
			},
		},
		{
			config: "a:b:debug",
			entries: []testEntry{
				{"a:b", logger2.LevelDebug, "test", nil, nil},
			},
			wantResult: "- debug [                 a:b] test\n",
		},
		{
			config: "a:b:debug",
			entries: []testEntry{
				{"a:b", logger2.LevelDebug, "test", nil, nil},
			},
			mockWriterErr: errTest,
			wantErr:       fmt.Errorf("log write error: %w", errTest),
		},
		{
			config: "a:b:debug",
			entries: []testEntry{
				{"a:b", logger2.LevelDebug, "test", nil, nil},
			},
			mockFormatterErr: errTest,
			wantErr:          fmt.Errorf("log format error: %w", errTest),
		},
		{
			config: "a:b:debug",
			ctx: logger2.Ctx{
				"k1": "v1",
				"k2": "v2",
			},
			entries: []testEntry{
				{"a:b", logger2.LevelDebug, "test", nil, logger2.Ctx{"k2": "v3"}},
			},
			wantResult: "- debug [                 a:b] test k1=v1 k2=v3\n",
		},
		{
			config: "a:b:trace,c:d",
			ctx: logger2.Ctx{
				"k1": "v1",
				"k2": "v2",
			},
			entries: []testEntry{
				{"a:b", logger2.LevelTrace, "test1", nil, logger2.Ctx{"k2": "v3"}},
				{"a:b", logger2.LevelDebug, "test2", nil, logger2.Ctx{"k3": "v3"}},
				{"a:b", logger2.LevelInfo, "test3", nil, logger2.Ctx{"k4": "v4"}},
				{"a:b", logger2.LevelWarn, "test4", nil, logger2.Ctx{"k5": "v5"}},
				{"a:b", logger2.LevelError, "", errTest, logger2.Ctx{"k6": "v6"}},
				{"c:d", logger2.LevelTrace, "test1", nil, logger2.Ctx{"k2": "v3"}},
				{"c:d", logger2.LevelDebug, "test2", nil, logger2.Ctx{"k3": "v3"}},
				{"c:d", logger2.LevelInfo, "test3", nil, logger2.Ctx{"k4": "v4"}},
				{"c:d", logger2.LevelWarn, "test4", nil, logger2.Ctx{"k5": "v5"}},
				{"c:d", logger2.LevelError, "", errTest, logger2.Ctx{"k6": "v6"}},
				{"e:f", logger2.LevelTrace, "test1", nil, logger2.Ctx{"k2": "v3"}},
				{"e:f", logger2.LevelDebug, "test2", nil, logger2.Ctx{"k3": "v3"}},
				{"e:f", logger2.LevelInfo, "test3", nil, logger2.Ctx{"k4": "v4"}},
				{"e:f", logger2.LevelWarn, "test4", nil, logger2.Ctx{"k5": "v5"}},
				{"e:f", logger2.LevelError, "", errTest, logger2.Ctx{"k6": "v6"}},
			},
			wantResult: `- trace [                 a:b] test1 k1=v1 k2=v3
- debug [                 a:b] test2 k1=v1 k2=v2 k3=v3
-  info [                 a:b] test3 k1=v1 k2=v2 k4=v4
-  warn [                 a:b] test4 k1=v1 k2=v2 k5=v5
- error [                 a:b] test err k1=v1 k2=v2 k6=v6
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

		root := logger2.New().WithConfig(logger2.NewConfigMapFromString(tc.config)).
			WithWriter(w).
			WithCtx(tc.ctx).
			WithFormatter(formatter)

		w.mockErr = tc.mockWriterErr
		formatter.mockErr = tc.mockFormatterErr

		var gotErr error

		for _, entry := range tc.entries {
			switch entry.level {
			case logger2.LevelError:
				_, gotErr = root.WithNamespace(entry.namespace).Error(entry.err, entry.ctx)
			case logger2.LevelWarn:
				_, gotErr = root.WithNamespace(entry.namespace).Warn(entry.message, entry.ctx)
			case logger2.LevelInfo:
				_, gotErr = root.WithNamespace(entry.namespace).Info(entry.message, entry.ctx)
			case logger2.LevelDebug:
				_, gotErr = root.WithNamespace(entry.namespace).Debug(entry.message, entry.ctx)
			case logger2.LevelTrace:
				_, gotErr = root.WithNamespace(entry.namespace).Trace(entry.message, entry.ctx)
			case logger2.LevelDisabled:
				fallthrough
			case logger2.LevelUnknown:
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

	log := logger2.New().WithConfig(logger2.LevelInfo).
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

	log := logger2.New()

	assert.Equal(t, logger2.Ctx(nil), log.Ctx())
	assert.Equal(t, logger2.Ctx{"a": "b"}, log.WithCtx(logger2.Ctx{"a": "b"}).Ctx())
}

func BenchmarkLogger_Disabled(b *testing.B) {
	log := logger2.New().WithNamespace("test").WithConfig(logger2.LevelDisabled)

	var thread int64

	b.RunParallel(func(pb *testing.PB) {
		curThread := atomic.AddInt64(&thread, 1)

		var n int64

		for pb.Next() {
			curN := atomic.AddInt64(&n, 1)
			_, _ = log.Info("benchmark", logger2.Ctx{"thread": curThread, "n": curN})
		}
	})
}

func BenchmarkLogger_Enabled(b *testing.B) {
	log := logger2.New().WithNamespace("test").WithConfig(logger2.LevelInfo)

	var thread int64

	b.RunParallel(func(pb *testing.PB) {
		curThread := atomic.AddInt64(&thread, 1)

		var n int64

		for pb.Next() {
			curN := atomic.AddInt64(&n, 1)
			_, _ = log.Info("benchmark", logger2.Ctx{"thread": curThread, "n": curN})
		}
	})
}

func BenchmarkLogger_EnabledWithoutSorting(b *testing.B) {
	log := logger2.New().WithNamespace("test").WithConfig(logger2.LevelInfo).
		WithFormatter(logger2.NewStringFormatter(logger2.StringFormatterParams{
			DateLayout:               "",
			DisableContextKeySorting: true,
		}))

	var thread int64

	b.RunParallel(func(pb *testing.PB) {
		curThread := atomic.AddInt64(&thread, 1)

		var n int64

		for pb.Next() {
			curN := atomic.AddInt64(&n, 1)
			_, _ = log.Info("benchmark", logger2.Ctx{"thread": curThread, "n": curN})
		}
	})
}
