package services

import (
	"context"
	"errors"
	"task-pool-system.com/task-pool-system/internal/constants"

	"github.com/redis/rueidis"

	model "task-pool-system.com/task-pool-system/internal/models"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

type TaskService struct {
	repo          *repository.TaskRepository
	pool          *PoolService
	redis         rueidis.Client
	redisQueueKey string
}

var ErrTaskQueueFull = errors.New("task queue is full")

func NewTaskService(
	redis rueidis.Client,
	repo *repository.TaskRepository,
	pool *PoolService,
	redisQueueKey string,
) *TaskService {
	return &TaskService{
		repo:          repo,
		pool:          pool,
		redis:         redis,
		redisQueueKey: redisQueueKey,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, title, description string) (*model.Task, error) {
	acquiredToken, err := s.acquireQueueToken(ctx)
	if err != nil {
		return nil, err
	}

	task, err := s.createTaskRecord(ctx, title, description, acquiredToken)
	if err != nil {
		return nil, err
	}

	if err := s.enqueueTaskForProcessing(ctx, task, acquiredToken); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) acquireQueueToken(ctx context.Context) (bool, error) {
	if s.redis == nil || s.redisQueueKey == "" {
		return false, nil
	}

	if err := s.redis.Do(
		ctx,
		s.redis.B().Lpop().Key(s.redisQueueKey).Build(),
	).Error(); err != nil {
		if rueidis.IsRedisNil(err) {
			return false, ErrTaskQueueFull
		}

		return false, err
	}

	return true, nil
}

func (s *TaskService) createTaskRecord(
	ctx context.Context,
	title,
	description string,
	acquiredToken bool,
) (*model.Task, error) {
	task, err := s.repo.CreateTask(ctx, title, description)
	if err != nil {
		if acquiredToken {
			s.releaseQueueToken(ctx)
		}

		return nil, err
	}

	return task, nil
}

func (s *TaskService) enqueueTaskForProcessing(
	ctx context.Context,
	task *model.Task,
	acquiredToken bool,
) error {
	if ok := s.pool.Enqueue(task.ID); ok {
		return nil
	}

	task.Status = constants.StatusFailed
	_ = s.repo.Update(ctx, task)

	if acquiredToken {
		s.releaseQueueToken(ctx)
	}

	return ErrTaskQueueFull
}

func (s *TaskService) releaseQueueToken(ctx context.Context) {
	if s.redis == nil || s.redisQueueKey == "" {
		return
	}

	_ = s.redis.Do(
		ctx,
		s.redis.B().Rpush().Key(s.redisQueueKey).Element("1").Build(),
	).Error()
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context) ([]model.Task, error) {
	return s.repo.List(ctx)
}
