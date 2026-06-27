package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/new-api-tools/backend/internal/logger"
)

// Handler executes one task iteration.
type Handler func(ctx context.Context) error

// NextInterval calculates the interval before the next iteration.
type NextInterval func(ctx context.Context) (time.Duration, error)

// Task defines a managed background task.
type Task struct {
	Name         string
	Interval     time.Duration
	InitialDelay time.Duration
	Timeout      time.Duration
	Handler      Handler
	NextInterval NextInterval

	mu             sync.RWMutex
	running        bool
	started        bool
	lastStartedAt  time.Time
	lastFinishedAt time.Time
	lastDuration   time.Duration
	lastErr        string
	runCount       int64
	successCount   int64
	skipCount      int64
}

// TaskStatus is a serializable task status snapshot.
type TaskStatus struct {
	Name            string `json:"name"`
	IntervalSeconds int64  `json:"interval_seconds"`
	Running         bool   `json:"running"`
	Started         bool   `json:"started"`
	LastStartedAt   string `json:"last_started_at,omitempty"`
	LastFinishedAt  string `json:"last_finished_at,omitempty"`
	LastDurationMS  int64  `json:"last_duration_ms"`
	LastErr         string `json:"last_err,omitempty"`
	RunCount        int64  `json:"run_count"`
	SuccessCount    int64  `json:"success_count"`
	SkipCount       int64  `json:"skip_count"`
}

// Manager coordinates background task lifecycle.
type Manager struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.RWMutex
	tasks   map[string]*Task
	started bool

	warmup *WarmupTracker
}

var (
	managerOnce sync.Once
	manager     *Manager
)

// GetManager returns the singleton task manager.
func GetManager() *Manager {
	managerOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		manager = &Manager{
			ctx:    ctx,
			cancel: cancel,
			tasks:  make(map[string]*Task),
			warmup: NewWarmupTracker(),
		}
	})
	return manager
}

// Register adds a periodic task.
func (m *Manager) Register(task *Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.Handler == nil {
		return fmt.Errorf("task handler is required: %s", task.Name)
	}
	if task.Interval <= 0 && task.NextInterval == nil {
		return fmt.Errorf("task interval is required: %s", task.Name)
	}
	if task.Timeout <= 0 {
		task.Timeout = 5 * time.Minute
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return fmt.Errorf("task manager already started")
	}
	if _, exists := m.tasks[task.Name]; exists {
		return fmt.Errorf("task already registered: %s", task.Name)
	}
	m.tasks[task.Name] = task
	return nil
}

// Start starts all registered tasks once.
func (m *Manager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	m.mu.Unlock()

	m.warmup.Start()
	m.warmup.MarkStep("后台任务启动", "success", fmt.Sprintf("已注册 %d 个后台任务", len(tasks)))
	for _, task := range tasks {
		m.wg.Add(1)
		go m.runTaskLoop(task)
	}
	logger.L.Task(fmt.Sprintf("后台任务管理器已启动，任务数: %d", len(tasks)))
}

// Stop cancels tasks and waits until they exit.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.started = false
	m.mu.Unlock()

	m.cancel()
	m.wg.Wait()
	logger.L.Task("后台任务管理器已停止")
}

// GetStatus returns snapshots of all tasks.
func (m *Manager) GetStatus() []TaskStatus {
	m.mu.RLock()
	tasks := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	m.mu.RUnlock()

	statuses := make([]TaskStatus, 0, len(tasks))
	for _, task := range tasks {
		statuses = append(statuses, task.status())
	}
	return statuses
}

// GetWarmupStatus returns current warmup status.
func (m *Manager) GetWarmupStatus() WarmupStatus {
	return m.warmup.Status()
}

func (m *Manager) runTaskLoop(task *Task) {
	defer m.wg.Done()

	if task.InitialDelay > 0 {
		if !sleepContext(m.ctx, task.InitialDelay) {
			return
		}
	}

	for {
		m.executeTask(task)

		next := task.Interval
		if task.NextInterval != nil {
			ctx, cancel := context.WithTimeout(m.ctx, minDuration(task.Timeout, 10*time.Second))
			interval, err := task.NextInterval(ctx)
			cancel()
			if err != nil {
				logger.L.TaskError(fmt.Sprintf("任务 %s 计算下次间隔失败: %v", task.Name, err))
			} else if interval > 0 {
				next = interval
			}
		}
		if next <= 0 {
			next = time.Minute
		}
		if !sleepContext(m.ctx, next) {
			return
		}
	}
}

func (m *Manager) executeTask(task *Task) {
	if !task.tryStart() {
		return
	}
	startedAt := time.Now()
	defer func() {
		if r := recover(); r != nil {
			task.finish(startedAt, fmt.Errorf("panic: %v", r))
			logger.L.TaskError(fmt.Sprintf("任务 %s panic: %v", task.Name, r))
		}
	}()

	ctx, cancel := context.WithTimeout(m.ctx, task.Timeout)
	defer cancel()
	err := task.Handler(ctx)
	task.finish(startedAt, err)
	if err != nil {
		logger.L.TaskError(fmt.Sprintf("任务 %s 执行失败: %v", task.Name, err))
	}
}

func (t *Task) tryStart() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.running {
		t.skipCount++
		return false
	}
	t.running = true
	t.started = true
	t.lastStartedAt = time.Now()
	return true
}

func (t *Task) finish(startedAt time.Time, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.running = false
	t.lastFinishedAt = time.Now()
	t.lastDuration = t.lastFinishedAt.Sub(startedAt)
	t.runCount++
	if err != nil {
		t.lastErr = err.Error()
		return
	}
	t.lastErr = ""
	t.successCount++
}

func (t *Task) status() TaskStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	status := TaskStatus{
		Name:            t.Name,
		IntervalSeconds: int64(t.Interval.Seconds()),
		Running:         t.running,
		Started:         t.started,
		LastDurationMS:  t.lastDuration.Milliseconds(),
		LastErr:         t.lastErr,
		RunCount:        t.runCount,
		SuccessCount:    t.successCount,
		SkipCount:       t.skipCount,
	}
	if !t.lastStartedAt.IsZero() {
		status.LastStartedAt = t.lastStartedAt.Format(time.RFC3339)
	}
	if !t.lastFinishedAt.IsZero() {
		status.LastFinishedAt = t.lastFinishedAt.Format(time.RFC3339)
	}
	return status
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}
