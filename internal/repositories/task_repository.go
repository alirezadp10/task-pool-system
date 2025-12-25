package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-pool-system.com/task-pool-system/internal/constants"
	apperrors "task-pool-system.com/task-pool-system/internal/errors"
	model "task-pool-system.com/task-pool-system/internal/models"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) CreateTask(ctx context.Context, title, description string) (*model.Task, error) {
	task := &model.Task{
		ID:          uuid.NewString(),
		Title:       title,
		Description: description,
		Status:      constants.StatusPending,
		Version:     1,
		CreatedAt:   time.Now().UTC(),
	}

	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}

	return task, nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id string) (*model.Task, error) {
	var task model.Task
	err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrTaskNotFound
		}
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) List(ctx context.Context) ([]model.Task, error) {
	var tasks []model.Task
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) ListPendingUnstarted(ctx context.Context, limit int) ([]model.Task, error) {
	if limit <= 0 {
		return nil, apperrors.ErrInvalidLimit
	}

	var tasks []model.Task
	query := r.db.WithContext(ctx).
		Where("status = ? AND started_at IS NULL", constants.StatusPending).
		Order("created_at asc").Limit(limit)

	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	res := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("id = ? AND version = ?", task.ID, task.Version).
		Updates(map[string]interface{}{
			"title":        task.Title,
			"description":  task.Description,
			"status":       task.Status,
			"duration_sec": task.DurationSec,
			"started_at":   task.StartedAt,
			"completed_at": task.CompletedAt,
			"version":      gorm.Expr("version + 1"),
		})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return apperrors.ErrOptimisticLock
	}

	task.Version++
	return nil
}
