package rbd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func floatPtr(value float64) *float64 {
	return &value
}

func TestTheAlarmRingsOnlyWhenTheFloorIsFirstBroken(t *testing.T) {
	assert.True(t, FellBelowFloor(floatPtr(0.96), 0.94, 0.95))
	assert.True(t, FellBelowFloor(nil, 0.94, 0.95))
	assert.False(t, FellBelowFloor(floatPtr(0.93), 0.94, 0.95))
	assert.False(t, FellBelowFloor(floatPtr(0.96), 0.97, 0.95))
	assert.False(t, FellBelowFloor(nil, 0.95, 0.95))
}
