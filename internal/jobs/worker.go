package jobs

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Runner interface {
	Kind() string
	Run(ctx context.Context, job Job) (string, error)
}

type Queue interface {
	Claim(ctx context.Context) (*Job, error)
	Finish(ctx context.Context, jobId, result string) error
	Fail(ctx context.Context, jobId, message string) error
	ReleaseStranded(ctx context.Context, olderThanMinutes int) (int64, error)
}

type Worker struct {
	queue     Queue
	runners   map[string]Runner
	idle      time.Duration
	stall     int
	sweepp    time.Duration
	lastSweep time.Time
}

func NewWorker(queue Queue, idle time.Duration, runners ...Runner) *Worker {
	registered := map[string]Runner{}
	for _, runner := range runners {
		registered[runner.Kind()] = runner
	}
	if idle <= 0 {
		idle = 2 * time.Second
	}
	return &Worker{queue: queue, runners: registered, idle: idle, stall: 30, sweepp: 5 * time.Minute}
}

func (w *Worker) Kinds() []string {
	kinds := make([]string, 0, len(w.runners))
	for kind := range w.runners {
		kinds = append(kinds, kind)
	}
	return kinds
}

func (w *Worker) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		w.sweep(ctx)
		if w.once(ctx) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(w.idle):
		}
	}
}

func (w *Worker) sweep(ctx context.Context) {
	if time.Since(w.lastSweep) < w.sweepp {
		return
	}
	w.lastSweep = time.Now()
	released, err := w.queue.ReleaseStranded(ctx, w.stall)
	if err != nil {
		log.Printf("could not look for stranded jobs: %v", err)
		return
	}
	if released > 0 {
		log.Printf("released %d stranded job(s) back to the queue", released)
	}
}

func (w *Worker) once(ctx context.Context) bool {
	job, err := w.queue.Claim(ctx)
	if err != nil {
		log.Printf("could not take a job from the queue: %v", err)
		return false
	}
	if job == nil {
		return false
	}
	runner, known := w.runners[job.Kind]
	if !known {
		w.queue.Fail(ctx, job.JobId, "Nothing here knows how to run a "+job.Kind+" job.")
		return true
	}
	started := time.Now()
	result, err := w.safely(ctx, runner, *job)
	if err != nil {
		log.Printf("%s job %s gave up after %s: %v", job.Kind, job.JobId, time.Since(started).Round(time.Millisecond), err)
		w.queue.Fail(ctx, job.JobId, err.Error())
		return true
	}
	log.Printf("%s job %s finished in %s", job.Kind, job.JobId, time.Since(started).Round(time.Millisecond))
	if err := w.queue.Finish(ctx, job.JobId, result); err != nil {
		log.Printf("finished %s but could not record it: %v", job.JobId, err)
	}
	return true
}

func (w *Worker) safely(ctx context.Context, runner Runner, job Job) (result string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("The job stopped unexpectedly: %v", recovered)
		}
	}()
	return runner.Run(ctx, job)
}
