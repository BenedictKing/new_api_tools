package tasks

import (
	"sync"
	"time"
)

// WarmupStep describes one startup warmup step.
type WarmupStep struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
}

// WarmupStatus is consumed by the frontend warmup screen.
type WarmupStatus struct {
	Status      string       `json:"status"`
	Progress    int          `json:"progress"`
	Message     string       `json:"message"`
	Steps       []WarmupStep `json:"steps"`
	StartedAt   string       `json:"started_at,omitempty"`
	CompletedAt string       `json:"completed_at,omitempty"`
}

// WarmupTracker tracks lightweight startup warmup progress.
type WarmupTracker struct {
	mu          sync.RWMutex
	status      string
	message     string
	startedAt   time.Time
	completedAt time.Time
	steps       []WarmupStep
}

// NewWarmupTracker creates the default warmup tracker.
func NewWarmupTracker() *WarmupTracker {
	return &WarmupTracker{
		status:  "pending",
		message: "系统正在初始化",
		steps: []WarmupStep{
			{Name: "后台任务启动", Status: "pending"},
			{Name: "模型列表缓存", Status: "pending"},
			{Name: "Token 分组缓存", Status: "pending"},
			{Name: "Analytics Summary 缓存", Status: "pending"},
		},
	}
}

// Start marks warmup as initializing.
func (w *WarmupTracker) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == "ready" {
		return
	}
	w.status = "initializing"
	w.message = "系统正在预热缓存"
	w.startedAt = time.Now()
}

// MarkStep updates one step state.
func (w *WarmupTracker) MarkStep(name, status, message string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	for i := range w.steps {
		if w.steps[i].Name != name {
			continue
		}
		w.steps[i].Status = status
		w.steps[i].Message = message
		if status == "running" && w.steps[i].StartedAt == "" {
			w.steps[i].StartedAt = now
		}
		if status == "success" || status == "error" || status == "skipped" {
			w.steps[i].EndedAt = now
		}
		return
	}
}

// Ready marks warmup as completed.
func (w *WarmupTracker) Ready(message string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status = "ready"
	w.message = message
	w.completedAt = time.Now()
}

// Status returns a snapshot of the current warmup state.
func (w *WarmupTracker) Status() WarmupStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	steps := make([]WarmupStep, len(w.steps))
	copy(steps, w.steps)
	progress := 0
	if len(steps) > 0 {
		completed := 0
		for _, step := range steps {
			if step.Status == "success" || step.Status == "error" || step.Status == "skipped" {
				completed++
			}
		}
		progress = completed * 100 / len(steps)
	}
	if w.status == "ready" {
		progress = 100
	}
	status := WarmupStatus{
		Status:   w.status,
		Progress: progress,
		Message:  w.message,
		Steps:    steps,
	}
	if !w.startedAt.IsZero() {
		status.StartedAt = w.startedAt.Format(time.RFC3339)
	}
	if !w.completedAt.IsZero() {
		status.CompletedAt = w.completedAt.Format(time.RFC3339)
	}
	return status
}
