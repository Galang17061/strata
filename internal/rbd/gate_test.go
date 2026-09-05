package rbd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheGateLetsOneInAndTurnsTheNextAwayPolitely(t *testing.T) {
	gate := NewGate(1, 30*time.Millisecond)
	leave, err := gate.Enter(context.Background())
	require.NoError(t, err)

	_, err = gate.Enter(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "The studio is full")

	leave()
	leaveAgain, err := gate.Enter(context.Background())
	require.NoError(t, err)
	leaveAgain()
}

func TestAWaitingCallerGetsInWhenASlotFrees(t *testing.T) {
	gate := NewGate(1, 500*time.Millisecond)
	leave, err := gate.Enter(context.Background())
	require.NoError(t, err)
	go func() {
		time.Sleep(50 * time.Millisecond)
		leave()
	}()
	leaveSecond, err := gate.Enter(context.Background())
	require.NoError(t, err)
	leaveSecond()
}

func TestACancelledCallerStopsWaiting(t *testing.T) {
	gate := NewGate(1, time.Minute)
	leave, err := gate.Enter(context.Background())
	require.NoError(t, err)
	defer leave()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err = gate.Enter(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "called off")
}

func TestAMissingGateNeverBlocksAnyone(t *testing.T) {
	var gate *Gate
	leave, err := gate.Enter(context.Background())
	require.NoError(t, err)
	leave()
}
