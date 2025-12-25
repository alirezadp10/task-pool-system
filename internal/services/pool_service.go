package services

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"task-pool-system.com/task-pool-system/internal/constants"
	apperrors "task-pool-system.com/task-pool-system/internal/errors"
	model "task-pool-system.com/task-pool-system/internal/models"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

type PoolService struct {
	repo               *repository.TaskRepository
	queue              chan string
	pollIntervalSec    int
	pollBatchSize      int
	wg                 sync.WaitGroup
	pollerWg           sync.WaitGroup
	shutdownPollerChan chan struct{}
}

func NewPoolService(
	repo *repository.TaskRepository,
	workers int,
	queueSize int,
	pollIntervalSec int,
	pollBatchSize int,
) *PoolService {
	p := &PoolService{
		repo:               repo,
		queue:              make(chan string, queueSize),
		pollIntervalSec:    pollIntervalSec,
		pollBatchSize:      pollBatchSize,
		shutdownPollerChan: make(chan struct{}),
	}

	for i := 1; i <= workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	p.pollerWg.Add(1)
	go p.pollTasks()

	return p
}

func (p *PoolService) worker(workerID int) {
	defer p.wg.Done()

	log.Printf("worker %d started", workerID)

	for taskID := range p.queue {
		p.handleTask(workerID, taskID)
	}

	log.Printf("worker %d stopped", workerID)
}

func (p *PoolService) handleTask(workerID int, taskID string) {
	ctx := context.Background()

	task, err := p.updateTaskStatus(ctx, workerID, taskID)
	if err != nil {
		return
	}

	log.Printf("worker %d processing task %s", workerID, taskID)

	p.simulateWork()

	if err := p.completeTask(ctx, workerID, task); err != nil {
		return
	}
}

func (p *PoolService) updateTaskStatus(ctx context.Context, workerID int, taskID string) (*model.Task, error) {
	task, err := p.repo.FindByID(ctx, taskID)
	if err != nil {
		log.Printf("worker %d: task %s not found", workerID, taskID)
		return nil, err
	}

	startedAt := time.Now().UTC()
	task.Status = constants.StatusInProgress
	task.StartedAt = &startedAt

	if err := p.repo.Update(ctx, task); err != nil {
		if errors.Is(err, apperrors.ErrOptimisticLock) {
			log.Printf("worker %d: optimistic lock conflict starting task %s", workerID, taskID)
			return nil, err
		}
		log.Printf("worker %d: failed to update task %s", workerID, taskID)
		return nil, err
	}

	return task, nil
}

func (p *PoolService) simulateWork() {
	duration := time.Duration(rand.Intn(5)+1) * time.Second
	time.Sleep(duration)
}

func (p *PoolService) completeTask(ctx context.Context, workerID int, task *model.Task) error {
	completedAt := time.Now().UTC()
	task.Status = constants.StatusCompleted
	task.CompletedAt = &completedAt

	if err := p.repo.Update(ctx, task); err != nil {
		if errors.Is(err, apperrors.ErrOptimisticLock) {
			log.Printf("worker %d: optimistic lock conflict completing task %s", workerID, task.ID)
			return err
		}
		log.Printf("worker %d: failed to complete task %s", workerID, task.ID)
		return err
	}

	log.Printf("worker %d completed task %s", workerID, task.ID)
	return nil
}

func (p *PoolService) pollTasks() {
	log.Printf("task poller started (interval: %ds, batch size: %d)", p.pollIntervalSec, p.pollBatchSize)
	defer p.pollerWg.Done()

	ticker := time.NewTicker(time.Duration(p.pollIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.fetchAndEnqueueTasks()
		case <-p.shutdownPollerChan:
			log.Println("task poller stopped")
			return
		}
	}
}

func (p *PoolService) fetchAndEnqueueTasks() {
	ctx := context.Background()

	tasks, err := p.repo.ListPendingUnstarted(ctx, p.pollBatchSize)
	if err != nil {
		log.Printf("failed to fetch pending tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	log.Printf("polling found %d pending task(s)", len(tasks))

	for _, task := range tasks {
		select {
		case p.queue <- task.ID:
			log.Printf("enqueued task %s to worker pool", task.ID)
		default:
			log.Printf("queue full, skipping task %s (will retry next poll)", task.ID)
			return
		}
	}
}

func (p *PoolService) Shutdown(ctx context.Context) {
	close(p.shutdownPollerChan)
	p.pollerWg.Wait()
	log.Println("task poller shut down")

	close(p.queue)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("worker pool shut down cleanly")
	case <-ctx.Done():
		log.Println("worker pool shutdown timed out")
	}
}
