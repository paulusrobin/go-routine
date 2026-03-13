package goroutine_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/paulusrobin/go-routine"
)

func TestPoolGroup(t *testing.T) {
	p := goroutine.NewPool(2, 10)
	defer p.Stop()

	g := p.Group(context.Background())
	var counter atomic.Int32
	tasksCount := 10

	for range tasksCount {
		g.Go(func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			counter.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if counter.Load() != int32(tasksCount) {
		t.Errorf("expected %d tasks completed via Group, got %d", tasksCount, counter.Load())
	}
}

func TestPoolGroupError(t *testing.T) {
	p := goroutine.NewPool(2, 10)
	defer p.Stop()

	g := p.Group(context.Background())
	expectedErr := context.DeadlineExceeded

	g.Go(func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return expectedErr
	})

	g.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	err := g.Wait()
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestPoolGroupStopOnFirstError(t *testing.T) {
	p := goroutine.NewPool(4, 10)
	defer p.Stop()

	g := p.Group(context.Background())
	expectedErr := context.DeadlineExceeded
	var counter atomic.Int32

	// Task 1: Fails quickly
	g.Go(func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		return expectedErr
	})

	// Task 2: Slow task, should be cancelled
	g.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			counter.Add(1)
			return nil
		}
	})

	// Task 3: Another task submitted before the error happens
	g.Go(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	err := g.Wait()
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// Task 4: Submitted AFTER the error should return immediately or not be submitted
	g.Go(func(ctx context.Context) error {
		counter.Add(1)
		return nil
	})

	// Wait for any remaining tasks to settle (though they should have finished during g.Wait())
	// But Task 4 might still be in the pool if it was submitted before g.Wait() finished.
	// Actually, g.Wait() waits for all g.Go() calls to finish their g.wg.Done().

	if counter.Load() != 1 {
		t.Errorf("expected 1 successful tasks, got %d", counter.Load())
	}
}

func TestPoolGroupGoContextCancellation(t *testing.T) {
	// Pool with no queue and only 1 worker to make it easily "busy"
	p := goroutine.NewPool(1, 0)
	defer p.Stop()

	g := p.Group(context.Background())

	// Task 1: Occupy the only worker
	wg := make(chan struct{})
	g.Go(func(ctx context.Context) error {
		<-wg
		return nil
	})

	// Task 2: Will be blocked in Submit because worker 1 is busy and queue is 0.
	// We want to cancel the group while Task 2 is waiting.
	ctx, cancel := context.WithCancel(context.Background())
	g2 := p.Group(ctx)

	// Start a goroutine that will cancel g2's context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// This should block until g2's context is cancelled
	start := time.Now()
	g2.Go(func(ctx context.Context) error {
		return nil
	})
	duration := time.Since(start)

	if duration < 50*time.Millisecond {
		t.Errorf("expected Go to block for at least 50ms, took %v", duration)
	}

	err := g2.Wait()
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Clean up Task 1
	close(wg)
	g.Wait()
}

func TestPoolGroupParentContextCancel(t *testing.T) {
	p := goroutine.NewPool(2, 10)
	defer p.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	g := p.Group(ctx)

	g.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	})

	cancel()
	err := g.Wait()
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected error %v, got %v", context.Canceled, err)
	}
}

func TestPoolGroupPanic(t *testing.T) {
	p := goroutine.NewPool(2, 10)
	defer p.Stop()

	g := p.Group(context.Background())
	panicMsg := "test panic"

	g.Go(func(ctx context.Context) error {
		panic(panicMsg)
	})

	err := g.Wait()
	if err == nil {
		t.Errorf("expected error from panic, got nil")
	}

	if err.Error() != panicMsg {
		t.Errorf("expected panic error message %q, got %q", panicMsg, err.Error())
	}
}
