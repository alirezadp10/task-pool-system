package services

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	dto "task-pool-system.com/task-pool-system/internal/data_models"
	"time"

	"task-pool-system.com/task-pool-system/internal/constants"
	apperrors "task-pool-system.com/task-pool-system/internal/errors"
	model "task-pool-system.com/task-pool-system/internal/models"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

type PoolService struct {
	repo               *repository.TaskRepository
	queue              chan dto.TaskMessageData
	workerWg           sync.WaitGroup
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
		queue:              make(chan dto.TaskMessageData, queueSize),
		shutdownPollerChan: make(chan struct{}),
	}

	p.startWorkers(workers)

	p.startPoller(pollIntervalSec, pollBatchSize)

	return p
}

func (p *PoolService) startWorkers(workers int) {
	for i := 1; i <= workers; i++ {
		workerID := i
		p.workerWg.Add(1)
		go func() {
			defer p.workerWg.Done()
			for taskMsgData := range p.queue {
				p.handleTask(workerID, taskMsgData)
			}
		}()
	}
}

func (p *PoolService) startPoller(pollIntervalSec, pollBatchSize int) {
	p.pollerWg.Add(1)
	go p.pollTasks(pollIntervalSec, pollBatchSize)
}

func (p *PoolService) handleTask(workerID int, taskMsgData dto.TaskMessageData) {
	ctx := context.Background()

	task, err := p.startTask(ctx, workerID, taskMsgData)
	if err != nil {
		log.Printf("worker %d stopped with error %s", workerID, err.Error())
		return
	}

	p.simulateWork()

	if err := p.completeTask(ctx, workerID, task); err != nil {
		log.Printf("worker %d stopped with error %s", workerID, err.Error())
		return
	}
}

func (p *PoolService) startTask(ctx context.Context, workerID int, taskMsgData dto.TaskMessageData) (*model.Task, error) {
	task, err := p.repo.MarkAsInProgress(ctx, taskMsgData.TaskID, taskMsgData.TaskVersion)
	if err != nil {
		if errors.Is(err, apperrors.ErrOptimisticLock) {
			log.Printf("worker %d: task %s already processed or not found", workerID, taskMsgData.TaskID)
			return nil, err
		}
		log.Printf("worker %d: failed to mark task %s as in_progress", workerID, taskMsgData.TaskID)
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

	return nil
}

func (p *PoolService) pollTasks(pollIntervalSec, pollBatchSize int) {
	defer p.pollerWg.Done()

	ticker := time.NewTicker(time.Duration(pollIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.fetchAndEnqueueTasks(pollBatchSize)
		case <-p.shutdownPollerChan:
			return
		}
	}
}

func (p *PoolService) fetchAndEnqueueTasks(pollBatchSize int) {
	ctx := context.Background()

	tasks, err := p.repo.ListPendingUnstarted(ctx, pollBatchSize)
	if err != nil {
		log.Printf("failed to fetch pending tasks: %v", err)
		return
	}

	if len(tasks) == 0 {
		return
	}

	for _, task := range tasks {
		select {
		case p.queue <- dto.TaskMessageData{TaskID: task.ID, TaskVersion: task.Version}:
		default:
			return
		}
	}
}

func (p *PoolService) Shutdown(ctx context.Context) {
	close(p.shutdownPollerChan)
	p.pollerWg.Wait()

	close(p.queue)

	done := make(chan struct{})
	go func() {
		p.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("worker pool shut down cleanly")
	case <-ctx.Done():
		log.Println("worker pool shutdown timed out")
	}
}
