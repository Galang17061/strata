package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeQueue struct {
	waiting   []Job
	finished  map[string]string
	failed    map[string]string
	released  int
	claimErr  error
	sweepErr  error
	sweptWith []int
}

func newFakeQueue(waiting ...Job) *fakeQueue {
	return &fakeQueue{waiting: waiting, finished: map[string]string{}, failed: map[string]string{}}
}

func (q *fakeQueue) Claim(context.Context) (*Job, error) {
	if q.claimErr != nil {
		return nil, q.claimErr
	}
	if len(q.waiting) == 0 {
		return nil, nil
	}
	job := q.waiting[0]
	q.waiting = q.waiting[1:]
	return &job, nil
}

func (q *fakeQueue) Finish(_ context.Context, jobId, result string) error {
	q.finished[jobId] = result
	return nil
}

func (q *fakeQueue) Fail(_ context.Context, jobId, message string) error {
	q.failed[jobId] = message
	return nil
}

func (q *fakeQueue) ReleaseStranded(_ context.Context, minutes int) (int64, error) {
	q.sweptWith = append(q.sweptWith, minutes)
	if q.sweepErr != nil {
		return 0, q.sweepErr
	}
	return int64(q.released), nil
}

type fakeRunner struct {
	kind    string
	answer  string
	failure error
	panics  bool
	seen    []Job
}

func (r *fakeRunner) Kind() string {
	return r.kind
}

func (r *fakeRunner) Run(_ context.Context, job Job) (string, error) {
	r.seen = append(r.seen, job)
	if r.panics {
		panic("the maths fell over")
	}
	return r.answer, r.failure
}

func TestWorkerHandsAJobToTheRunnerThatKnowsIt(t *testing.T) {
	queue := newFakeQueue(Job{JobId: "a", Kind: "simulation", Request: "{}"})
	runner := &fakeRunner{kind: "simulation", answer: `{"ok":true}`}
	worker := NewWorker(queue, time.Millisecond, runner)
	assert.True(t, worker.once(context.Background()))
	assert.Equal(t, `{"ok":true}`, queue.finished["a"])
	assert.Len(t, runner.seen, 1)
	assert.Empty(t, queue.failed)
}

func TestWorkerFailsAJobNobodyKnowsHowToRun(t *testing.T) {
	queue := newFakeQueue(Job{JobId: "b", Kind: "weather"})
	worker := NewWorker(queue, time.Millisecond, &fakeRunner{kind: "simulation"})
	assert.True(t, worker.once(context.Background()))
	assert.Contains(t, queue.failed["b"], "weather")
	assert.Empty(t, queue.finished)
}

func TestWorkerSurvivesARunnerThatPanics(t *testing.T) {
	queue := newFakeQueue(Job{JobId: "c", Kind: "simulation"})
	worker := NewWorker(queue, time.Millisecond, &fakeRunner{kind: "simulation", panics: true})
	assert.True(t, worker.once(context.Background()))
	assert.Contains(t, queue.failed["c"], "stopped unexpectedly")
}

func TestWorkerRecordsWhyARunnerGaveUp(t *testing.T) {
	queue := newFakeQueue(Job{JobId: "d", Kind: "simulation"})
	worker := NewWorker(queue, time.Millisecond, &fakeRunner{kind: "simulation", failure: errors.New("no components")})
	assert.True(t, worker.once(context.Background()))
	assert.Equal(t, "no components", queue.failed["d"])
}

func TestWorkerRestsWhenTheQueueIsEmpty(t *testing.T) {
	queue := newFakeQueue()
	worker := NewWorker(queue, time.Millisecond, &fakeRunner{kind: "simulation"})
	assert.False(t, worker.once(context.Background()))
}

func TestWorkerSweepsForStrandedJobsOnceEveryWhile(t *testing.T) {
	queue := newFakeQueue()
	queue.released = 2
	worker := NewWorker(queue, time.Millisecond, &fakeRunner{kind: "simulation"})
	worker.sweep(context.Background())
	worker.sweep(context.Background())
	require.Len(t, queue.sweptWith, 1)
	assert.Equal(t, 30, queue.sweptWith[0])
	worker.lastSweep = time.Now().Add(-10 * time.Minute)
	worker.sweep(context.Background())
	assert.Len(t, queue.sweptWith, 2)
}

func TestWorkerStopsWhenTheContextIsDone(t *testing.T) {
	queue := newFakeQueue()
	worker := NewWorker(queue, time.Millisecond, &fakeRunner{kind: "simulation"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("the worker did not stop when it was told to")
	}
}

func TestWorkerNamesTheKindsItCanTake(t *testing.T) {
	worker := NewWorker(newFakeQueue(), time.Millisecond, &fakeRunner{kind: "simulation"})
	assert.Equal(t, []string{"simulation"}, worker.Kinds())
}
