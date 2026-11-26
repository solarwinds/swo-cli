package main

import (
	"context"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalExitCode(t *testing.T) {
	assert.Equal(t, 130, signalExitCode(syscall.SIGINT))
	assert.Equal(t, 130, signalExitCode(os.Interrupt))
	assert.Equal(t, 143, signalExitCode(syscall.SIGTERM))
}

func TestNotifyContextReportsSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sending signals to the current process is not supported on Windows")
	}

	ctx, received, stop := notifyContext(context.Background(), syscall.SIGTERM)
	t.Cleanup(stop)
	assert.Nil(t, received())

	p, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, p.Signal(syscall.SIGTERM))

	select {
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("context was not cancelled by SIGTERM")
	}
	assert.Equal(t, syscall.SIGTERM, received())
	assert.Equal(t, 143, signalExitCode(received()))
}

func TestNotifyContextStopWithoutSignal(t *testing.T) {
	ctx, received, stop := notifyContext(context.Background(), syscall.SIGTERM)
	stop()
	stop()

	assert.ErrorIs(t, ctx.Err(), context.Canceled)
	assert.Nil(t, received())
}
