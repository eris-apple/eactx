package eactx

import (
	"context"
	"sync"
	"time"
)

type CancelFunc context.CancelFunc
type DoneFunc func()
type TimeoutFunc func()

// State - the state of the context lifecycle.
type State int

const (
	Created State = iota
	Running
	Canceled
	Deadlined
	Finished
)

// Context — a wrapper for context.
type Context struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	onDone    []func()
	onCancel  []func()
	onTimeout []func()

	state State
}

// startMonitoring — converts the state to "State.Running" and monitors the completion of the context.
func (cw *Context) startMonitoring() {
	cw.setState(Running)
	<-cw.ctx.Done()

	if err := cw.ctx.Err(); err != nil {
		switch err {
		case context.Canceled:
			cw.setState(Canceled)
			for _, f := range cw.onCancel {
				f()
			}
		case context.DeadlineExceeded:
			cw.setState(Deadlined)
			for _, f := range cw.onTimeout {
				f()
			}
		default:
			cw.setState(Finished)
		}
	}
	for _, f := range cw.onDone {
		f()
	}
}

// OnDone — adds a callback that is called when the context ends.
func (cw *Context) OnDone(f func()) {
	cw.onDone = append(cw.onDone, f)
}

// OnCancel — adds a callback that is called when the context is canceled.
func (cw *Context) OnCancel(f func()) {
	cw.onCancel = append(cw.onCancel, f)
}

// OnTimeout — adds a callback that is called when the context time expires.
func (cw *Context) OnTimeout(f func()) {
	cw.onTimeout = append(cw.onTimeout, f)
}

// Done — returns the context termination channel.
func (cw *Context) Done() <-chan struct{} {
	return cw.ctx.Done()
}

// IsDone — returns the execution status of the context.
func (cw *Context) IsDone() bool {
	return cw.state != Created && cw.state != Running
}

// Cancel — causes context cancellation.
func (cw *Context) Cancel() {
	cw.cancel()
}

// GetCancelFunc — returns the cancel function.
func (cw *Context) GetCancelFunc() context.CancelFunc {
	return cw.cancel
}

// Err — returns a context error.
func (cw *Context) Err() error {
	return cw.ctx.Err()
}

// Value — returns the value stored in the context.
func (cw *Context) Value(key interface{}) interface{} {
	return cw.ctx.Value(key)
}

// State — returns the current state of the context lifecycle.
func (cw *Context) State() State {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return cw.state
}

// setState — sets a new lifecycle state.
func (cw *Context) setState(state State) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.state = state
}

// GetContext — returns the original context.
func (cw *Context) GetContext() context.Context {
	return cw.ctx
}

// NewContextWithCancel — creates a new Context with the ability to cancel.
func NewContextWithCancel(parent context.Context) *Context {
	ctx, cancel := context.WithCancel(parent)
	cw := &Context{
		ctx:    ctx,
		cancel: cancel,
		state:  Created,
	}
	go cw.startMonitoring()
	return cw
}

// NewContextWithTimeout — creates a new Context with a timeout.
func NewContextWithTimeout(parent context.Context, timeout time.Duration) *Context {
	ctx, cancel := context.WithTimeout(parent, timeout)
	cw := &Context{
		ctx:    ctx,
		cancel: cancel,
		state:  Created,
	}
	go cw.startMonitoring()
	return cw
}
