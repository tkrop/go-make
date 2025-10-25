package sys_test

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tkrop/go-make/internal/sys"
)

func TestSignaler(t *testing.T) {
	done := make(chan struct{})
	var signaled os.Signal
	var cancelled bool

	handler := func(cancel context.CancelFunc, sig os.Signal) {
		assert.NotNil(t, cancel)
		cancelled = true
		signaled = sig
		close(done)
	}

	signaler := sys.NewSignaler(handler, syscall.SIGUSR1)
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

	assert.True(t, cancelled)
	assert.Equal(t, syscall.SIGUSR1, signaled)
	assert.NotNil(t, ctx.Done())
}
