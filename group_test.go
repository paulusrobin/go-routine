package goroutine_test

import (
	"context"
	"errors"
	"fmt"
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

func TestGroup(t *testing.T) {
	p := goroutine.NewPool(10, 20)
	defer p.Stop()

	g := p.Group(context.Background())
	tasksCount := 10
	for idx := range tasksCount {
		g.Go(func(ctx context.Context) error {
			fmt.Println("start", idx+1, "task")
			time.Sleep(10 * time.Millisecond)
			if idx+1 == 5 {
				fmt.Println("error", idx+1, "task")
				return fmt.Errorf("5 is error")
			}
			fmt.Println("finish", idx+1, "task")
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Println(err.Error())
	}
}
