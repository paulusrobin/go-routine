package goroutine

import (
	"context"
	"fmt"
	"sync"
)

// Group represents a group of tasks submitted to the pool.
type Group struct {
	pool   *Pool
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelCauseFunc
}

// Group creates a new group for running tasks in the pool.
func (p *Pool) Group(ctx context.Context) *Group {
	ctx, cancel := context.WithCancelCause(ctx)
	return &Group{
		pool:   p,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Go submits a task to the pool and tracks it in the group.
func (g *Group) Go(fn Task) {
	if g.ctx.Err() != nil {
		return
	}
	g.wg.Add(1)

	task := func(ctx context.Context) error {
		defer g.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				var err error
				if e, ok := r.(error); ok {
					err = e
				} else {
					err = fmt.Errorf("%v", r)
				}
				g.cancel(err)
			}
		}()
		if err := fn(g.ctx); err != nil {
			g.cancel(err)
			return err
		}
		return nil
	}

	err := g.pool.SubmitCtx(g.ctx, task)
	if err != nil {
		g.wg.Done()
	}
}

// Wait waits for all tasks in the group to finish.
func (g *Group) Wait() error {
	g.wg.Wait()
	return context.Cause(g.ctx)
}
