package goroutine_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/paulusrobin/go-routine"
)

func TestPool(t *testing.T) {
	workers := 5
	queueSize := 10
	p := goroutine.NewPool(workers, queueSize)
	defer p.Stop()

	var counter atomic.Int32
	tasksCount := 20
	var wg sync.WaitGroup
	wg.Add(tasksCount)

	for range tasksCount {
		err := p.Submit(func(ctx context.Context) error {
			defer wg.Done()
			counter.Add(1)
			return nil
		})
		if err != nil {
			t.Errorf("failed to submit task: %v", err)
		}
	}

	wg.Wait()
	if counter.Load() != int32(tasksCount) {
		t.Errorf("expected %d tasks completed, got %d", tasksCount, counter.Load())
	}
}

func TestPoolWorkerLimit(t *testing.T) {
	workers := 2
	p := goroutine.NewPool(workers, 0)
	defer p.Stop()

	var activeWorkers atomic.Int32
	var maxActive atomic.Int32
	var wg sync.WaitGroup
	wg.Add(4)

	for range 4 {
		p.Submit(func(ctx context.Context) error {
			defer wg.Done()
			current := activeWorkers.Add(1)
			for {
				max := maxActive.Load()
				if current <= max || maxActive.CompareAndSwap(max, current) {
					break
				}
			}
			time.Sleep(100 * time.Millisecond)
			activeWorkers.Add(-1)
			return nil
		})
	}

	wg.Wait()
	if maxActive.Load() > int32(workers) {
		t.Errorf("expected max %d workers, got %d", workers, maxActive.Load())
	}
}

func TestPoolStop(t *testing.T) {
	p := goroutine.NewPool(1, 10)
	var counter atomic.Int32

	for range 5 {
		p.Submit(func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
			return nil
		})
	}

	p.Stop()

	if counter.Load() != 5 {
		t.Errorf("expected all 5 submitted tasks to finish after Stop(), got %d", counter.Load())
	}

	err := p.Submit(func(ctx context.Context) error { return nil })
	if err != goroutine.ErrPoolStopped {
		t.Errorf("expected ErrPoolStopped, got %v", err)
	}
}

func TestPoolContext(t *testing.T) {
	p := goroutine.NewPool(1, 1)
	ctx := p.Context()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	select {
	case <-ctx.Done():
		t.Error("context should not be canceled yet")
	default:
	}

	p.Stop()

	select {
	case <-ctx.Done():
	default:
		t.Error("context should be canceled after Stop()")
	}
}

func TestPoolTaskContext(t *testing.T) {
	p := goroutine.NewPool(1, 1)
	defer p.Stop()

	poolCtx := p.Context()
	var taskCtx context.Context
	var wg sync.WaitGroup
	wg.Add(1)

	p.Submit(func(ctx context.Context) error {
		defer wg.Done()
		taskCtx = ctx
		return nil
	})

	wg.Wait()
	if taskCtx != poolCtx {
		t.Error("expected task to receive pool context")
	}
}
