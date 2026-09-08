package jobs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Galang17061/strata-api/internal/domain"
)

func TestListenerTellsEveryWatcher(t *testing.T) {
	listener := NewListener("")
	first, stopFirst := listener.Watch()
	second, stopSecond := listener.Watch()
	defer stopFirst()
	defer stopSecond()
	assert.Equal(t, 2, listener.Watchers())
	listener.deliver("hello")
	assert.Equal(t, "hello", <-first)
	assert.Equal(t, "hello", <-second)
}

func TestListenerForgetsAWatcherThatLeaves(t *testing.T) {
	listener := NewListener("")
	_, stop := listener.Watch()
	stop()
	assert.Equal(t, 0, listener.Watchers())
	listener.deliver("nobody is listening")
}

func TestListenerDropsWordsARestingWatcherCannotTake(t *testing.T) {
	listener := NewListener("")
	updates, stop := listener.Watch()
	defer stop()
	for i := 0; i < 100; i++ {
		listener.deliver("flood")
	}
	assert.Equal(t, 16, len(updates))
}

func TestJobViewCarriesRequestAndResultAsJSON(t *testing.T) {
	result := `{"reliability":0.98}`
	job := Job{
		JobId:     "a",
		Kind:      "simulation",
		Status:    StatusDone,
		Request:   `{"trials":100}`,
		Result:    &result,
		CreatedAt: domain.Now(),
	}
	view := job.View()
	assert.JSONEq(t, `{"trials":100}`, string(view.Request))
	assert.JSONEq(t, result, string(view.Result))
	encoded, err := json.Marshal(view)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"reliability":0.98`)
}

func TestJobViewLeavesRubbishOut(t *testing.T) {
	rubbish := "not json at all"
	job := Job{JobId: "b", Request: "{", Result: &rubbish}
	view := job.View()
	assert.Nil(t, view.Request)
	assert.Nil(t, view.Result)
}
