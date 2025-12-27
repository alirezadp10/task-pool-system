package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"task-pool-system.com/task-pool-system/internal/constants"
	excptions "task-pool-system.com/task-pool-system/internal/exceptions"
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
			return nil, excptions.ErrTaskNotFound
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
		return nil, excptions.ErrInvalidLimit
	}

	var tasks []model.Task
	now := time.Now().UTC()

	// SQLite-compatible query using subquery and RETURNING (SQLite 3.35.0+)
	err := r.db.WithContext(ctx).Raw(`
        UPDATE tasks
        SET status = ?, started_at = ?, version = version + 1
        WHERE id IN (
            SELECT id FROM tasks
            WHERE status = ?
            ORDER BY created_at
            LIMIT ?
        )
        RETURNING *;
    `, constants.StatusInProgress, now, constants.StatusPending, limit).Scan(&tasks).Error

	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) MarkAsComplete(ctx context.Context, taskID string, taskVersion uint) error {
	res := r.db.WithContext(ctx).Model(&model.Task{}).
		Where("id = ? AND version = ?", taskID, taskVersion).
		Updates(map[string]interface{}{
			"status":       constants.StatusCompleted,
			"completed_at": time.Now().UTC(),
			"version":      gorm.Expr("version + 1"),
		})

	if res.Error != nil {
		return res.Error
	}

	if res.RowsAffected == 0 {
		return excptions.ErrOptimisticLock
	}

	return nil
}
