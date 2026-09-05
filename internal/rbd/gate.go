package rbd

import (
	"context"
	"strconv"
	"time"

	"github.com/Galang17061/strata-api/internal/domain"
)

type Gate struct {
	slots chan struct{}
	wait  time.Duration
}

func NewGate(capacity int, wait time.Duration) *Gate {
	if capacity < 1 {
		capacity = 1
	}
	return &Gate{slots: make(chan struct{}, capacity), wait: wait}
}

func (g *Gate) Enter(ctx context.Context) (func(), error) {
	if g == nil {
		return func() {}, nil
	}
	select {
	case g.slots <- struct{}{}:
		return func() { <-g.slots }, nil
	default:
	}
	timer := time.NewTimer(g.wait)
	defer timer.Stop()
	select {
	case g.slots <- struct{}{}:
		return func() { <-g.slots }, nil
	case <-timer.C:
		return nil, domain.InvalidOperation("The studio is full: " + strconv.Itoa(cap(g.slots)) + " searches are already running. Give it a moment and run again.")
	case <-ctx.Done():
		return nil, domain.InvalidOperation("The search was called off while waiting for a free slot.")
	}
}
