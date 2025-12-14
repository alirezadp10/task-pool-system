package services

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/redis/rueidis"

	"task-pool-system.com/task-pool-system/internal/constants"
	model "task-pool-system.com/task-pool-system/internal/models"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

type PoolService struct {
	queue         chan string
	wg            sync.WaitGroup
	requeueWG     sync.WaitGroup
	enqueued      sync.Map
	repo          *repository.TaskRepository
	redis         rueidis.Client
	redisQueueKey string
	requeueStop   chan struct{}
}

func NewPoolService(
	repo *repository.TaskRepository,
	workers int,
	queueSize int,
	redis rueidis.Client,
	redisQueueKey string,
) *PoolService {
	p := &PoolService{
		queue:         make(chan string, queueSize),
		repo:          repo,
		redis:         redis,
		redisQueueKey: redisQueueKey,
		requeueStop:   make(chan struct{}),
	}

	p.requeueWG.Add(1)
	go p.requeuePendingLoop()

	for i := 1; i <= workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	return p
}

func (p *PoolService) Enqueue(taskID string) bool {
	ok, _ := p.enqueueIfNotPresent(taskID)
	return ok
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
	defer p.releaseQueueToken(ctx, workerID)
	defer p.untrackEnqueued(taskID)

	task, err := p.findAndStartTask(ctx, workerID, taskID)
	if err != nil {
		return
	}

	log.Printf("worker %d processing task %s", workerID, taskID)

	p.simulateWork()

	if err := p.completeTask(ctx, workerID, task); err != nil {
		return
	}
}

func (p *PoolService) findAndStartTask(ctx context.Context, workerID int, taskID string) (*model.Task, error) {
	task, err := p.repo.FindByID(ctx, taskID)
	if err != nil {
		log.Printf("worker %d: task %s not found", workerID, taskID)
		return nil, err
	}

	startedAt := time.Now().UTC()
	task.Status = constants.StatusInProgress
	task.StartedAt = &startedAt

	if err := p.repo.Update(ctx, task); err != nil {
		if errors.Is(err, repository.ErrOptimisticLock) {
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
		if errors.Is(err, repository.ErrOptimisticLock) {
			log.Printf("worker %d: optimistic lock conflict completing task %s", workerID, task.ID)
			return err
		}
		log.Printf("worker %d: failed to complete task %s", workerID, task.ID)
		return err
	}

	log.Printf("worker %d completed task %s", workerID, task.ID)
	return nil
}

func (p *PoolService) releaseQueueToken(ctx context.Context, workerID int) {
	if p.redis == nil || p.redisQueueKey == "" {
		return
	}

	if err := p.redis.Do(
		ctx,
		p.redis.B().Rpush().Key(p.redisQueueKey).Element("1").Build(),
	).Error(); err != nil {
		log.Printf("worker %d: failed to release redis queue token: %v", workerID, err)
	}
}

func (p *PoolService) requeuePendingLoop() {
	defer p.requeueWG.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.requeuePendingOnce()
		case <-p.requeueStop:
			return
		}
	}
}

func (p *PoolService) requeuePendingOnce() {
	ctx := context.Background()

	tasks, err := p.repo.ListPendingUnstarted(ctx, 50)
	if err != nil {
		log.Printf("requeue: failed to list pending tasks: %v", err)
		return
	}

	for _, task := range tasks {
		enqueued, queueFull := p.enqueueIfNotPresent(task.ID)
		if !enqueued && !queueFull {
			continue
		}
		if queueFull {
			return
		}
	}
}

func (p *PoolService) enqueueIfNotPresent(taskID string) (bool, bool) {
	if !p.trackEnqueued(taskID) {
		return false, false
	}

	select {
	case p.queue <- taskID:
		return true, false
	default:
		p.untrackEnqueued(taskID)
		return false, true
	}
}

func (p *PoolService) trackEnqueued(taskID string) bool {
	_, loaded := p.enqueued.LoadOrStore(taskID, struct{}{})
	return !loaded
}

func (p *PoolService) untrackEnqueued(taskID string) {
	p.enqueued.Delete(taskID)
}

func (p *PoolService) Shutdown(ctx context.Context) {
	close(p.requeueStop)
	p.requeueWG.Wait()
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
