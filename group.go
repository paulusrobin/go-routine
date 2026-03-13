package goroutine

import (
	"context"
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
	err := g.pool.Submit(func(ctx context.Context) error {
		defer g.wg.Done()
		if err := fn(g.ctx); err != nil {
			g.cancel(err)
			return err
		}
		return nil
	})
	if err != nil {
		g.wg.Done()
	}
}

// Wait waits for all tasks in the group to finish.
func (g *Group) Wait() error {
	g.wg.Wait()
	return context.Cause(g.ctx)
}
