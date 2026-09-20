package closer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

type testLogger struct{}

func (testLogger) Info(context.Context, string, ...zap.Field)  {}
func (testLogger) Error(context.Context, string, ...zap.Field) {}

func TestNewCreatesIndependentCloser(t *testing.T) {
	first := New(testLogger{}, time.Second)
	second := New(testLogger{}, time.Second)
	var firstCalls, secondCalls atomic.Int32
	first.Add(func(context.Context) error { firstCalls.Add(1); return nil })
	second.Add(func(context.Context) error { secondCalls.Add(1); return nil })

	if err := first.CloseAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if firstCalls.Load() != 1 || secondCalls.Load() != 0 {
		t.Fatalf("calls after first close = %d, %d", firstCalls.Load(), secondCalls.Load())
	}
	if err := second.CloseAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if secondCalls.Load() != 1 {
		t.Fatalf("second calls = %d", secondCalls.Load())
	}
}

func TestCloseAllRunsFunctionsOnlyOnce(t *testing.T) {
	c := New(testLogger{}, time.Second)
	var calls atomic.Int32
	c.Add(func(context.Context) error { calls.Add(1); return nil })

	if err := c.CloseAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.CloseAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
}
