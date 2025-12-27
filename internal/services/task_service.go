package services

import (
	"context"
	model "task-pool-system.com/task-pool-system/pkg/models"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

type TaskService struct {
	repo *repository.TaskRepository
	pool *PoolService
}

func NewTaskService(
	repo *repository.TaskRepository,
) *TaskService {
	return &TaskService{
		repo: repo,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, title, description string) (*model.Task, error) {
	return s.repo.CreateTask(ctx, title, description)
}

func (s *TaskService) GetTask(ctx context.Context, id string) (*model.Task, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context) ([]model.Task, error) {
	return s.repo.List(ctx)
}
