package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Job represents a unit of work to be executed by the worker pool
type Job struct {
	ID      string
	Execute func(ctx context.Context, logger *zap.Logger) error
	Result  chan error
}

// IWorkerPool defines the interface for the worker pool
type IWorkerPool interface {
	Submit(job Job) error
	Start()
	Stop()
	Size() int
}

type workerPool struct {
	size       int
	jobQueue   chan Job
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	mu         sync.RWMutex
}

// NewWorkerPool creates a new bounded worker pool
func NewWorkerPool(size int, queueSize int, logger *zap.Logger) IWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	if queueSize == 0 {
		queueSize = size * 2
	}

	return &workerPool{
		size:     size,
		jobQueue: make(chan Job, queueSize),
		logger:   logger,
		ctx:      ctx,
		cancel:   cancel,
		running:  false,
	}
}

func (p *workerPool) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	p.running = true
	p.logger.Info("starting worker pool", zap.Int("size", p.size))

	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *workerPool) Stop() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.mu.Unlock()

	p.logger.Info("stopping worker pool")
	p.cancel()
	close(p.jobQueue)
	p.wg.Wait()
	p.logger.Info("worker pool stopped")
}

func (p *workerPool) Size() int {
	return p.size
}

func (p *workerPool) Submit(job Job) error {
	p.mu.RLock()
	running := p.running
	p.mu.RUnlock()

	if !running {
		if job.Result != nil {
			job.Result <- context.Canceled
			close(job.Result)
		}
		return context.Canceled
	}

	select {
	case p.jobQueue <- job:
		return nil
	case <-p.ctx.Done():
		if job.Result != nil {
			job.Result <- p.ctx.Err()
			close(job.Result)
		}
		return p.ctx.Err()
	}
}

func (p *workerPool) worker(id int) {
	defer p.wg.Done()

	p.logger.Debug("worker started", zap.Int("worker_id", id))

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				p.logger.Debug("worker shutting down (queue closed)", zap.Int("worker_id", id))
				return
			}

			p.executeJob(id, job)

		case <-p.ctx.Done():
			p.logger.Debug("worker shutting down (context cancelled)", zap.Int("worker_id", id))
			return
		}
	}
}

func (p *workerPool) executeJob(workerID int, job Job) {
	logger := p.logger.With(
		zap.Int("worker_id", workerID),
		zap.String("job_id", job.ID),
	)

	logger.Debug("executing job")

	// Execute the job
	err := job.Execute(p.ctx, logger)

	// Send result back if result channel is provided
	if job.Result != nil {
		select {
		case job.Result <- err:
		default:
			logger.Warn("could not send job result - channel full or closed")
		}
		close(job.Result)
	}

	if err != nil {
		logger.Error("job execution failed", zap.Error(err))
	} else {
		logger.Debug("job completed successfully")
	}
}

// SubmitAndWait submits a job and waits for its completion
func SubmitAndWait(pool IWorkerPool, ctx context.Context, logger *zap.Logger, id string, fn func(ctx context.Context, logger *zap.Logger) error) error {
	resultChan := make(chan error, 1)

	job := Job{
		ID:      id,
		Execute: fn,
		Result:  resultChan,
	}

	if err := pool.Submit(job); err != nil {
		return err
	}

	select {
	case err := <-resultChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
