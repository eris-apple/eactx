package eactx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// State - the state of the context lifecycle.
type State int

const (
	Created State = iota
	Running
	Canceled
	Deadlined
	Finished
)

// Context a wrapper for context.
type Context struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	cond *sync.Cond
	done bool

	state State

	onDone    []func()
	onCancel  []func()
	onTimeout []func()
}

// startMonitoring transitions state to "Running" and monitors context completion.
func (cw *Context) startMonitoring() {
	cw.setState(Running)

	select {
	case <-cw.ctx.Done():
		var newState State
		var callbacks []func()

		cw.mu.Lock()
		switch err := cw.ctx.Err(); {
		case errors.Is(err, context.Canceled):
			newState = Canceled
			callbacks = cw.onCancel
		case errors.Is(err, context.DeadlineExceeded):
			newState = Deadlined
			callbacks = cw.onTimeout
		}
		cw.mu.Unlock()

		if newState != Running {
			cw.setState(newState)
			for _, f := range callbacks {
				f()
			}
		} else {
			cw.setState(Finished)
			for _, f := range cw.onDone {
				f()
			}
		}
	}

	cw.done = true
	cw.cond.Broadcast()
}

// OnDone adds a callback that is called when the context ends.
func (cw *Context) OnDone(f func()) {
	if f != nil {
		cw.onDone = append(cw.onDone, f)
	}
}

// OnCancel adds a callback that is called when the context is canceled.
func (cw *Context) OnCancel(f func()) {
	if f != nil {
		cw.onCancel = append(cw.onCancel, f)
	}
}

// OnTimeout adds a callback that is called when the context time expires.
func (cw *Context) OnTimeout(f func()) {
	if f != nil {
		cw.onTimeout = append(cw.onTimeout, f)
	}
}

// Done returns the context's done channel.
func (cw *Context) Done() <-chan struct{} {
	return cw.ctx.Done()
}

// IsDone returns true if the context has finished.
func (cw *Context) IsDone() bool {
	return cw.state != Created && cw.state != Running
}

// Cancel causes the context to be canceled.
func (cw *Context) Cancel() {
	cw.cancel()
}

// CancelWithWait causes the context to be canceled.
func (cw *Context) CancelWithWait() {
	cw.Cancel()
	cw.mu.Lock()
	for !cw.done {
		cw.cond.Wait()
	}
	cw.mu.Unlock()
}

// Err returns any error associated with the context.
func (cw *Context) Err() error {
	return cw.ctx.Err()
}

// Value returns a value stored in the context.
func (cw *Context) Value(key interface{}) interface{} {
	return cw.ctx.Value(key)
}

// State returns the current state of the context lifecycle.
func (cw *Context) State() State {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return cw.state
}

// setState sets a new lifecycle state.
func (cw *Context) setState(state State) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.state = state
}

// Reset resets the context to the initial "Created" state for reuse.
func (cw *Context) Reset(parent context.Context) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	ctx, cancel := context.WithCancel(parent)
	cw.ctx = ctx
	cw.cancel = cancel
	cw.state = Created
	cw.onDone = nil
	cw.onCancel = nil
	cw.onTimeout = nil
	cw.done = false
	go cw.startMonitoring()
}

// WithValue adds a key-value pair to the context.
func (cw *Context) WithValue(key, value interface{}) {
	cw.ctx = context.WithValue(cw.ctx, key, value)
}

// Clone creates a copy of the current context with the same state and callbacks.
func (cw *Context) Clone() *Context {
	newCtx := &Context{
		ctx:       cw.ctx,
		cancel:    cw.cancel,
		state:     cw.state,
		onDone:    append([]func(){}, cw.onDone...),
		onCancel:  append([]func(){}, cw.onCancel...),
		onTimeout: append([]func(){}, cw.onTimeout...),
	}
	return newCtx
}

// Deadline returns the time when the context will be canceled, if set.
func (cw *Context) Deadline() (time.Time, bool) {
	return cw.ctx.Deadline()
}

// String returns a string representation of the current context state.
func (cw *Context) String() string {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return fmt.Sprintf("Context state: %v", cw.state)
}

// GetContext returns a context.Context.
func (cw *Context) GetContext() context.Context {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return cw.ctx
}

// NewContextWithCancel creates a new Context with the ability to cancel.
func NewContextWithCancel(parent context.Context) *Context {
	ctx, cancel := context.WithCancel(parent)
	cw := &Context{
		ctx:    ctx,
		cancel: cancel,
		state:  Created,
	}

	cw.cond = sync.NewCond(&cw.mu)

	go cw.startMonitoring()
	return cw
}

// NewContextWithTimeout creates a new Context with a timeout.
func NewContextWithTimeout(parent context.Context, timeout time.Duration) *Context {
	ctx, cancel := context.WithTimeout(parent, timeout)
	cw := &Context{
		ctx:    ctx,
		cancel: cancel,
		state:  Created,
	}

	cw.cond = sync.NewCond(&cw.mu)

	go cw.startMonitoring()
	return cw
}
