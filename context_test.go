package eactx

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewContextWithCancel(t *testing.T) {
	parent := context.Background()
	ctx := NewContextWithCancel(parent)

	if ctx.State() != Created {
		t.Errorf("expected state to be %v, got %v", Created, ctx.State())
	}

	if ctx.IsDone() {
		t.Error("expected IsDone to be false at the start")
	}

	ctx.CancelWithWait()
	if ctx.State() != Canceled {
		t.Errorf("expected state to be %v after cancel, got %v", Canceled, ctx.State())
	}
}

func TestNewContextWithTimeout(t *testing.T) {
	parent := context.Background()
	timeout := 50 * time.Millisecond
	ctx := NewContextWithTimeout(parent, timeout)

	time.Sleep(2 * timeout)

	if ctx.State() != Deadlined {
		t.Errorf("expected state to be %v after timeout, got %v", Deadlined, ctx.State())
	}

	if !ctx.IsDone() {
		t.Error("expected IsDone to be true after timeout")
	}
}

func TestContextCallbacks(t *testing.T) {
	parent := context.Background()
	ctx := NewContextWithCancel(parent)

	var onCancelCalled int32
	ctx.OnCancel(func() { atomic.AddInt32(&onCancelCalled, 1) })

	ctx.CancelWithWait()

	if atomic.LoadInt32(&onCancelCalled) != 1 {
		t.Error("expected OnCancel callback to be called once")
	}
}

func TestContextReset(t *testing.T) {
	parent := context.Background()
	ctx := NewContextWithCancel(parent)

	ctx.CancelWithWait()
	if ctx.State() != Canceled {
		t.Errorf("expected state to be %v after cancel, got %v", Canceled, ctx.State())
	}

	ctx.Reset(parent)
	if ctx.State() != Created {
		t.Errorf("expected state to be %v after reset, got %v", Created, ctx.State())
	}

	ctx.CancelWithWait()
	if ctx.State() != Canceled {
		t.Errorf("expected state to be %v after cancel, got %v", Canceled, ctx.State())
	}
}

func TestContextWithValue(t *testing.T) {
	parent := context.Background()
	ctx := NewContextWithCancel(parent)
	key, value := "key", "value"

	ctx.WithValue(key, value)
	if ctx.Value(key) != value {
		t.Errorf("expected Value to be %v, got %v", value, ctx.Value(key))
	}
}

func TestContextClone(t *testing.T) {
	parent := context.Background()
	ctx := NewContextWithCancel(parent)

	ctx.OnDone(func() {})
	ctx.OnCancel(func() {})
	ctx.OnTimeout(func() {})

	clone := ctx.Clone()

	if clone.State() != ctx.State() {
		t.Errorf("expected clone state to be %v, got %v", ctx.State(), clone.State())
	}

	if len(clone.onDone) != len(ctx.onDone) || len(clone.onCancel) != len(ctx.onCancel) || len(clone.onTimeout) != len(ctx.onTimeout) {
		t.Error("expected clone callbacks to match original callbacks")
	}
}
