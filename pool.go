package goroutine

import (
	"context"
	"errors"
	"sync"
)

// Task represents a function to be executed by the pool.
type Task func(ctx context.Context) error

// Pool is a goroutine pool that limits the number of concurrent goroutines.
type Pool struct {
	tasks      chan Task
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	maxWorkers int
	once       sync.Once
}

var (
	ErrPoolStopped = errors.New("pool is stopped")
)

// NewPool creates a new goroutine pool with the specified number of workers and queue size.
func NewPool(workers int, queueSize int) *Pool {
	if workers <= 0 {
		workers = 1
	}
	if queueSize < 0 {
		queueSize = 0
	}

	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		tasks:      make(chan Task, queueSize),
		ctx:        ctx,
		cancel:     cancel,
		maxWorkers: workers,
	}

	for range workers {
		p.wg.Add(1)
		go p.worker()
	}

	return p
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			// Process remaining tasks in the queue before stopping
			for {
				select {
				case task, ok := <-p.tasks:
					if !ok {
						return
					}
					_ = task(p.ctx)
				default:
					return
				}
			}
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			_ = task(p.ctx)
		}
	}
}

// Submit adds a task to the pool. Returns ErrPoolStopped if the pool is stopped.
func (p *Pool) Submit(task Task) error {
	select {
	case <-p.ctx.Done():
		return ErrPoolStopped
	default:
		select {
		case <-p.ctx.Done():
			return ErrPoolStopped
		case p.tasks <- task:
			return nil
		}
	}
}

// Context returns the pool's context.
func (p *Pool) Context() context.Context {
	return p.ctx
}

// Stop stops the pool and waits for all workers to finish.
func (p *Pool) Stop() {
	p.once.Do(func() {
		p.cancel()
		close(p.tasks)
		p.wg.Wait()
	})
}
