package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"task-pool-system.com/task-pool-system/internal/constants"
	model "task-pool-system.com/task-pool-system/internal/models"
	"task-pool-system.com/task-pool-system/internal/queue"
	repository "task-pool-system.com/task-pool-system/internal/repositories"
)

// mockTokenManager is a simple in-memory token manager for testing
type mockTokenManager struct {
	mu     sync.Mutex
	tokens int
}

func newMockTokenManager(capacity int) *mockTokenManager {
	return &mockTokenManager{tokens: capacity}
}

func (m *mockTokenManager) AcquireToken(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tokens <= 0 {
		return queue.ErrNoTokenAvailable
	}
	m.tokens--
	return nil
}

func (m *mockTokenManager) ReleaseToken(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens++
	return nil
}

func (m *mockTokenManager) InitializeTokens(ctx context.Context, count int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens = count
	return nil
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&model.Task{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)

	return db
}

func TestTaskService_AddAndCheckStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	tokenManager := newMockTokenManager(10)
	pool := NewPoolService(tokenManager, repo, 0, 10)
	defer pool.Shutdown(context.Background())
	service := NewTaskService(tokenManager, repo, pool)

	ctx := context.Background()
	title := "Test Task"
	desc := "Test Description"

	task, err := service.CreateTask(ctx, title, desc)
	if err != nil {
		t.Errorf("failed to create task: %v", err)
	}

	if task.ID == "" {
		t.Error("expected task ID to be set")
	}

	fetchedTask, err := service.GetTask(ctx, task.ID)
	if err != nil {
		t.Errorf("failed to get task: %v", err)
	}

	if fetchedTask.Status != constants.StatusPending {
		t.Errorf("expected status %s, got %s", constants.StatusPending, fetchedTask.Status)
	}
}

func TestTaskService_ConcurrentSubmissions(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)
	tokenManager := newMockTokenManager(100)
	pool := NewPoolService(tokenManager, repo, 0, 100)
	defer pool.Shutdown(context.Background())
	service := NewTaskService(tokenManager, repo, pool)

	const concurrentCount = 50
	var wg sync.WaitGroup
	wg.Add(concurrentCount)

	errs := make(chan error, concurrentCount)

	for i := 0; i < concurrentCount; i++ {
		go func(idx int) {
			defer wg.Done()
			_, err := service.CreateTask(context.Background(), "Title", "Desc")
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent creation failed: %v", err)
	}

	tasks, _ := service.ListTasks(context.Background())
	if len(tasks) != concurrentCount {
		t.Errorf("expected %d tasks, got %d", concurrentCount, len(tasks))
	}
}

func TestPoolService_EnqueueAndProcess(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)

	tokenManager := newMockTokenManager(10)
	pool := NewPoolService(tokenManager, repo, 2, 10)
	defer pool.Shutdown(context.Background())

	ctx := context.Background()
	task, _ := repo.CreateTask(ctx, "Pool Task 1", "Testing pool enqueue")

	ok := pool.Enqueue(task.ID)
	if !ok {
		t.Error("failed to enqueue task")
	}

	deadline := time.Now().Add(10 * time.Second)
	var finalStatus constants.TaskStatus
	for time.Now().Before(deadline) {
		t, err := repo.FindByID(ctx, task.ID)
		if err == nil {
			finalStatus = t.Status
			if finalStatus == constants.StatusCompleted {
				break
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	if finalStatus != constants.StatusCompleted {
		t.Errorf("task should be completed, but is %s", finalStatus)
	}
}

func TestPoolService_ConcurrentEnqueue(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)

	const queueSize = 10
	tokenManager := newMockTokenManager(queueSize)
	pool := NewPoolService(tokenManager, repo, 0, queueSize)
	defer pool.Shutdown(context.Background())

	ctx := context.Background()
	const taskCount = 20
	var wg sync.WaitGroup
	wg.Add(taskCount)

	results := make(chan bool, taskCount)

	for i := 0; i < taskCount; i++ {
		go func(idx int) {
			defer wg.Done()
			task, _ := repo.CreateTask(ctx, "Title", "Desc")
			results <- pool.Enqueue(task.ID)
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	for res := range results {
		if res {
			successCount++
		}
	}

	if successCount != queueSize {
		t.Errorf("expected %d tasks to be successfully enqueued (queue size), got %d", queueSize, successCount)
	}
}

func TestPoolService_WorkerMultiThreading(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTaskRepository(db)

	const workerCount = 5
	tokenManager := newMockTokenManager(50)
	pool := NewPoolService(tokenManager, repo, workerCount, 50)
	defer pool.Shutdown(context.Background())

	ctx := context.Background()
	const taskCount = 10
	taskIDs := make([]string, taskCount)

	for i := 0; i < taskCount; i++ {
		task, _ := repo.CreateTask(ctx, "Title", "Desc")
		taskIDs[i] = task.ID
		pool.Enqueue(task.ID)
	}

	deadline := time.Now().Add(20 * time.Second)
	completed := false
	for time.Now().Before(deadline) {
		allDone := true
		for _, id := range taskIDs {
			task, _ := repo.FindByID(ctx, id)
			if task.Status != constants.StatusCompleted {
				allDone = false
				break
			}
		}
		if allDone {
			completed = true
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !completed {
		t.Error("not all tasks were completed within deadline")
	}
}
