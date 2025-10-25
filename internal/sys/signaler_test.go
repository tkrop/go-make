package sys_test

import (
	"context"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tkrop/go-make/internal/sys"
)

func TestSignaler(t *testing.T) {
	t.Parallel()

	// Given
	done := make(chan struct{})
	signaled := atomic.Value{}
	cancelled := atomic.Bool{}

	signaler := sys.NewSignaler(func(cancel context.CancelFunc, sig os.Signal) {
		assert.NotNil(t, cancel)
		cancelled.Store(true)
		signaled.Store(sig)
		close(done)
	}, syscall.SIGUSR1)

	// When
	ctx := signaler.Signal(context.Background())

	go func() {
		// Send SIGUSR1 to self after a short delay.
		time.Sleep(20 * time.Millisecond)
		proc, _ := os.FindProcess(os.Getpid())
		_ = proc.Signal(syscall.SIGUSR1)
	}()

	select {
	case <-done: // handler should be called.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for signal handler")
	}

	// Then
	assert.True(t, cancelled.Load())
	assert.Equal(t, syscall.SIGUSR1, signaled.Load())
	assert.NotNil(t, ctx.Done())
}

func TestSignalerDone(t *testing.T) {
	t.Parallel()

	// Given
	called := atomic.Bool{}
	signaler := sys.NewSignaler(func(_ context.CancelFunc, _ os.Signal) {
		called.Store(true) // should not be called in this test.
	}, syscall.SIGUSR1)
	ctx, cancel := context.WithCancel(context.Background())

	// When
	signaler.Signal(ctx)
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Then
	assert.False(t, called.Load())
}
