package closer

import (
	"context"
	"sync"
	"time"
)

// New returns Closer and executes goroutine that awaits for ctx.Done() then run close(...) method
func New(ctx context.Context, errFunc ErrFunc, timeout time.Duration) *Closer {
	c := &Closer{
		done:       make(chan struct{}, 1),
		closeFuncs: make([]CloseFunc, 0),
		errFunc:    errFunc,
	}

	go func() {
		<-ctx.Done()

		closeCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		c.close(closeCtx)
	}()

	return c
}

// Closer struct
type Closer struct {
	mu         sync.Mutex
	done       chan struct{}
	closeFuncs []CloseFunc
	errFunc    ErrFunc
}

// CloseFunc is called when closer receives ctx.Done()
type CloseFunc func(context.Context) error

// ErrFunc is called to handle error
type ErrFunc func(error)

// Add closer func
func (c *Closer) Add(f CloseFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closeFuncs = append(c.closeFuncs, f)
}

func (c *Closer) close(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	go func() {
		for _, f := range c.closeFuncs {
			if err := f(ctx); err != nil {
				c.errFunc(err)
			}
		}

		c.done <- struct{}{}
	}()

	select {
	case <-c.done:
		break
	case <-ctx.Done():
		c.errFunc(ctx.Err())
	}
}
