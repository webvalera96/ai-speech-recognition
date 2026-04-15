package queue

import (
	"context"
	"log/slog"
	"sync"

	"github.com/webvalera96/ai-speech-recognition/internal/config"
	"github.com/webvalera96/ai-speech-recognition/internal/services"
	"go.uber.org/fx"
)

const defaultBuffer = 64

// Queue is a buffered channel-based job queue.
type Queue struct {
	ch chan services.Job

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewQueue constructs the queue with a fixed buffer size.
func NewQueue() *Queue {
	return &Queue{ch: make(chan services.Job, defaultBuffer)}
}

// Enqueue implements services.JobQueue.
func (q *Queue) Enqueue(ctx context.Context, job services.Job) error {
	select {
	case q.ch <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Jobs exposes the channel for workers (adapter-internal).
func (q *Queue) Jobs() <-chan services.Job {
	return q.ch
}

var _ services.JobQueue = (*Queue)(nil)

// WorkerParams is Fx input for starting pool workers.
type WorkerParams struct {
	fx.In
	LC       fx.Lifecycle
	Queue    *Queue
	Workflow *services.WorkflowService
	Cfg      *config.Config
}

// RegisterWorkers starts goroutines that process jobs until shutdown.
func RegisterWorkers(p WorkerParams) {
	p.Queue.mu.Lock()
	wctx, cancel := context.WithCancel(context.Background())
	p.Queue.cancel = cancel
	p.Queue.mu.Unlock()

	p.LC.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			n := p.Cfg.WorkerPool
			if n <= 0 {
				n = 2
			}
			for i := 0; i < n; i++ {
				p.Queue.wg.Add(1)
				go func(id int) {
					defer p.Queue.wg.Done()
					for {
						select {
						case <-wctx.Done():
							return
						case job, ok := <-p.Queue.ch:
							if !ok {
								return
							}
							runJob(id, p.Workflow, job)
						}
					}
				}(i)
			}
			return nil
		},
		OnStop: func(_ context.Context) error {
			p.Queue.mu.Lock()
			if p.Queue.cancel != nil {
				p.Queue.cancel()
			}
			p.Queue.mu.Unlock()
			p.Queue.wg.Wait()
			return nil
		},
	})
}

func runJob(workerID int, wf *services.WorkflowService, job services.Job) {
	ctx := context.Background()
	var err error
	switch job.Type {
	case services.JobTranscribe:
		err = wf.ProcessTranscribe(ctx, job)
	case services.JobChat:
		err = wf.ProcessChat(ctx, job)
	default:
		slog.Error("unknown job type", "worker", workerID, "type", job.Type)
		return
	}
	if err != nil {
		slog.Error("job failed", "worker", workerID, "type", job.Type, "err", err)
		_ = wf.NotifyError(ctx, job.ChatID, "Sorry, something went wrong processing your request.")
	}
}
