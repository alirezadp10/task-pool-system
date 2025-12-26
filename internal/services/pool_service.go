package services

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	dto "task-pool-system.com/task-pool-system/internal/data_models"
	exception "task-pool-system.com/task-pool-system/internal/exceptions"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
	"time"
)

type PoolService struct {
	ctx                context.Context
	repo               *repository.TaskRepository
	queue              chan dto.TaskMessageData
	workerWg           sync.WaitGroup
	pollerWg           sync.WaitGroup
	shutdownPollerChan chan struct{}
}

func NewPoolService(
	ctx context.Context,
	repo *repository.TaskRepository,
	workers int,
	queueSize int,
	pollIntervalSec int,
	pollBatchSize int,
) *PoolService {
	p := &PoolService{
		ctx:                ctx,
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
	// task stuck issue
	p.simulateWork()

	if err := p.completeTask(workerID, taskMsgData); err != nil {
		log.Printf("PoolService[handleTask]: worker %d stopped with error %s", workerID, err.Error())
		return
	}
}

func (p *PoolService) simulateWork() {
	duration := time.Duration(rand.Intn(5)+1) * time.Second
	time.Sleep(duration)
}

func (p *PoolService) completeTask(workerID int, taskMsgData dto.TaskMessageData) error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	err := p.repo.MarkAsComplete(ctx, taskMsgData.TaskID, taskMsgData.TaskVersion)
	if err != nil {
		if errors.Is(err, exception.ErrOptimisticLock) {
			log.Printf("PoolService[completeTask]: worker %d: failed to complete task %s", workerID, taskMsgData.TaskID)
			return err
		}
		log.Printf("PoolService[completeTask]: worker %d: failed to complete task %s", workerID, taskMsgData.TaskID)
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
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()

	free := cap(p.queue) - len(p.queue)
	if free == 0 {
		return
	}

	tasks, err := p.repo.ListPendingUnstarted(ctx, minInt(free, pollBatchSize))
	if err != nil {
		log.Printf("PoolService[fetchAndEnqueueTasks]: failed to claim pending tasks: %v", err)
		return
	}
	if len(tasks) == 0 {
		return
	}

	for _, task := range tasks {
		p.queue <- dto.TaskMessageData{TaskID: task.ID, TaskVersion: task.Version}
	}
}

func (p *PoolService) Shutdown() {
	close(p.shutdownPollerChan)
	p.pollerWg.Wait()

	close(p.queue)
	p.workerWg.Wait()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
