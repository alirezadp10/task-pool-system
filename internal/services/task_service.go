package services

import (
	"context"
	"errors"

	"task-pool-system.com/task-pool-system/internal/constants"
	model "task-pool-system.com/task-pool-system/internal/models"
	"task-pool-system.com/task-pool-system/internal/queue"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

type TaskService struct {
	repo         *repository.TaskRepository
	pool         *PoolService
	tokenManager queue.TokenManager
}

var ErrTaskQueueFull = errors.New("task queue is full")

func NewTaskService(
	tokenManager queue.TokenManager,
	repo *repository.TaskRepository,
	pool *PoolService,
) *TaskService {
	return &TaskService{
		repo:         repo,
		pool:         pool,
		tokenManager: tokenManager,
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
	if err := s.tokenManager.AcquireToken(ctx); err != nil {
		if errors.Is(err, queue.ErrNoTokenAvailable) {
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
	_ = s.tokenManager.ReleaseToken(ctx)
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context) ([]model.Task, error) {
	return s.repo.List(ctx)
}
