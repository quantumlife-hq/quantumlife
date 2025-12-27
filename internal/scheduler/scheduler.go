// Package scheduler provides task scheduling capabilities.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Scheduler manages scheduled tasks
type Scheduler struct {
	tasks    map[string]*Task
	running  map[string]context.CancelFunc
	mu       sync.RWMutex
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	started  bool
	timezone *time.Location
}

// Config configures the scheduler
type Config struct {
	Timezone string // Timezone for scheduling (default: Local)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Timezone: "Local",
	}
}

// NewScheduler creates a new scheduler
func NewScheduler(cfg Config) (*Scheduler, error) {
	tz, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		tz = time.Local
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		tasks:    make(map[string]*Task),
		running:  make(map[string]context.CancelFunc),
		ctx:      ctx,
		cancel:   cancel,
		timezone: tz,
	}, nil
}

// Task represents a scheduled task
type Task struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Schedule    Schedule      `json:"schedule"`
	Handler     TaskHandler   `json:"-"`
	Enabled     bool          `json:"enabled"`
	LastRun     *time.Time    `json:"last_run,omitempty"`
	NextRun     *time.Time    `json:"next_run,omitempty"`
	RunCount    int64         `json:"run_count"`
	ErrorCount  int64         `json:"error_count"`
	LastError   string        `json:"last_error,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	Timeout     time.Duration `json:"timeout"`
}

// TaskHandler is the function executed for a task
type TaskHandler func(ctx context.Context) error

// Schedule defines when a task runs
type Schedule struct {
	Type     ScheduleType `json:"type"`
	Interval time.Duration `json:"interval,omitempty"` // For interval schedules
	Cron     string       `json:"cron,omitempty"`     // For cron schedules
	At       string       `json:"at,omitempty"`       // For daily schedules (e.g., "08:00")
	Days     []time.Weekday `json:"days,omitempty"`   // For weekly schedules
}

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleInterval ScheduleType = "interval" // Run every X duration
	ScheduleDaily    ScheduleType = "daily"    // Run at specific time daily
	ScheduleWeekly   ScheduleType = "weekly"   // Run on specific days
	ScheduleCron     ScheduleType = "cron"     // Cron expression
	ScheduleOnce     ScheduleType = "once"     // Run once at specific time
)

// Register adds a task to the scheduler
func (s *Scheduler) Register(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}

	if task.Handler == nil {
		return fmt.Errorf("task handler is required")
	}

	if task.Timeout == 0 {
		task.Timeout = 5 * time.Minute
	}

	task.CreatedAt = time.Now()
	task.Enabled = true

	// Calculate next run
	nextRun := s.calculateNextRun(task.Schedule)
	task.NextRun = &nextRun

	s.tasks[task.ID] = task

	// Start task if scheduler is running
	if s.started {
		s.startTask(task)
	}

	return nil
}

// Unregister removes a task from the scheduler
func (s *Scheduler) Unregister(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if cancel, ok := s.running[taskID]; ok {
		cancel()
		delete(s.running, taskID)
	}

	delete(s.tasks, taskID)
	return nil
}

// Enable enables a task
func (s *Scheduler) Enable(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Enabled = true
	if s.started {
		s.startTask(task)
	}

	return nil
}

// Disable disables a task
func (s *Scheduler) Disable(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Enabled = false
	if cancel, ok := s.running[taskID]; ok {
		cancel()
		delete(s.running, taskID)
	}

	return nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("scheduler already started")
	}

	s.started = true

	// Start all enabled tasks
	for _, task := range s.tasks {
		if task.Enabled {
			s.startTask(task)
		}
	}

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.cancel()

	// Cancel all running tasks
	for _, cancel := range s.running {
		cancel()
	}
	s.running = make(map[string]context.CancelFunc)

	s.wg.Wait()
	s.started = false

	// Create new context for potential restart
	s.ctx, s.cancel = context.WithCancel(context.Background())

	return nil
}

// startTask starts a single task's scheduler loop
func (s *Scheduler) startTask(task *Task) {
	taskCtx, cancel := context.WithCancel(s.ctx)
	s.running[task.ID] = cancel

	s.wg.Add(1)
	go s.runTaskLoop(taskCtx, task)
}

// runTaskLoop is the main loop for a task
func (s *Scheduler) runTaskLoop(ctx context.Context, task *Task) {
	defer s.wg.Done()

	for {
		// Calculate wait duration
		var waitDuration time.Duration

		s.mu.RLock()
		if task.NextRun != nil {
			waitDuration = time.Until(*task.NextRun)
		} else {
			waitDuration = s.calculateNextRun(task.Schedule).Sub(time.Now())
		}
		s.mu.RUnlock()

		// Ensure minimum wait
		if waitDuration < 0 {
			waitDuration = 0
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(waitDuration):
			// Execute task
			s.executeTask(ctx, task)
		}

		// Check if one-time task
		if task.Schedule.Type == ScheduleOnce {
			return
		}
	}
}

// executeTask executes a single task
func (s *Scheduler) executeTask(ctx context.Context, task *Task) {
	// Create timeout context
	execCtx, cancel := context.WithTimeout(ctx, task.Timeout)
	defer cancel()

	// Update last run
	now := time.Now()
	s.mu.Lock()
	task.LastRun = &now
	task.RunCount++
	s.mu.Unlock()

	// Execute handler
	err := task.Handler(execCtx)

	// Update status
	s.mu.Lock()
	if err != nil {
		task.ErrorCount++
		task.LastError = err.Error()
	} else {
		task.LastError = ""
	}

	// Calculate next run
	nextRun := s.calculateNextRun(task.Schedule)
	task.NextRun = &nextRun
	s.mu.Unlock()
}

// calculateNextRun calculates the next run time for a schedule
func (s *Scheduler) calculateNextRun(schedule Schedule) time.Time {
	now := time.Now().In(s.timezone)

	switch schedule.Type {
	case ScheduleInterval:
		return now.Add(schedule.Interval)

	case ScheduleDaily:
		// Parse time
		hour, minute := 8, 0 // Default 8:00 AM
		fmt.Sscanf(schedule.At, "%d:%d", &hour, &minute)

		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, s.timezone)
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}
		return next

	case ScheduleWeekly:
		// Find next matching day
		hour, minute := 8, 0
		fmt.Sscanf(schedule.At, "%d:%d", &hour, &minute)

		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, s.timezone)

		// Check if any day matches
		for i := 0; i < 8; i++ {
			checkDay := next.Add(time.Duration(i) * 24 * time.Hour)
			for _, day := range schedule.Days {
				if checkDay.Weekday() == day && checkDay.After(now) {
					return checkDay
				}
			}
		}
		// Fallback to next week
		return next.Add(7 * 24 * time.Hour)

	case ScheduleOnce:
		// Parse the time from At field
		t, err := time.Parse(time.RFC3339, schedule.At)
		if err != nil {
			return now.Add(time.Minute)
		}
		return t

	default:
		return now.Add(time.Hour)
	}
}

// RunNow executes a task immediately
func (s *Scheduler) RunNow(taskID string) error {
	s.mu.RLock()
	task, ok := s.tasks[taskID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	go s.executeTask(s.ctx, task)
	return nil
}

// GetTask returns a task by ID
func (s *Scheduler) GetTask(taskID string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[taskID]
	return task, ok
}

// ListTasks returns all tasks
func (s *Scheduler) ListTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		Started:      s.started,
		TotalTasks:   len(s.tasks),
		RunningTasks: len(s.running),
		Timezone:     s.timezone.String(),
	}

	for _, task := range s.tasks {
		if task.Enabled {
			stats.EnabledTasks++
		}
		stats.TotalRuns += task.RunCount
		stats.TotalErrors += task.ErrorCount
	}

	return stats
}

// Stats contains scheduler statistics
type Stats struct {
	Started      bool   `json:"started"`
	TotalTasks   int    `json:"total_tasks"`
	EnabledTasks int    `json:"enabled_tasks"`
	RunningTasks int    `json:"running_tasks"`
	TotalRuns    int64  `json:"total_runs"`
	TotalErrors  int64  `json:"total_errors"`
	Timezone     string `json:"timezone"`
}

// Common task builders

// IntervalTask creates a task that runs at a fixed interval
func IntervalTask(id, name string, interval time.Duration, handler TaskHandler) *Task {
	return &Task{
		ID:       id,
		Name:     name,
		Schedule: Schedule{Type: ScheduleInterval, Interval: interval},
		Handler:  handler,
	}
}

// DailyTask creates a task that runs daily at a specific time
func DailyTask(id, name, at string, handler TaskHandler) *Task {
	return &Task{
		ID:       id,
		Name:     name,
		Schedule: Schedule{Type: ScheduleDaily, At: at},
		Handler:  handler,
	}
}

// WeeklyTask creates a task that runs on specific days
func WeeklyTask(id, name, at string, days []time.Weekday, handler TaskHandler) *Task {
	return &Task{
		ID:       id,
		Name:     name,
		Schedule: Schedule{Type: ScheduleWeekly, At: at, Days: days},
		Handler:  handler,
	}
}

// OnceTask creates a task that runs once at a specific time
func OnceTask(id, name string, at time.Time, handler TaskHandler) *Task {
	return &Task{
		ID:       id,
		Name:     name,
		Schedule: Schedule{Type: ScheduleOnce, At: at.Format(time.RFC3339)},
		Handler:  handler,
	}
}

// TaskBuilder provides fluent API for building tasks
type TaskBuilder struct {
	task *Task
}

// NewTask creates a new task builder
func NewTask(id string) *TaskBuilder {
	return &TaskBuilder{
		task: &Task{
			ID:      id,
			Enabled: true,
			Timeout: 5 * time.Minute,
		},
	}
}

// Name sets the task name
func (b *TaskBuilder) Name(name string) *TaskBuilder {
	b.task.Name = name
	return b
}

// Description sets the task description
func (b *TaskBuilder) Description(desc string) *TaskBuilder {
	b.task.Description = desc
	return b
}

// Every sets an interval schedule
func (b *TaskBuilder) Every(interval time.Duration) *TaskBuilder {
	b.task.Schedule = Schedule{Type: ScheduleInterval, Interval: interval}
	return b
}

// Daily sets a daily schedule
func (b *TaskBuilder) Daily(at string) *TaskBuilder {
	b.task.Schedule = Schedule{Type: ScheduleDaily, At: at}
	return b
}

// Weekly sets a weekly schedule
func (b *TaskBuilder) Weekly(at string, days ...time.Weekday) *TaskBuilder {
	b.task.Schedule = Schedule{Type: ScheduleWeekly, At: at, Days: days}
	return b
}

// Once sets a one-time schedule
func (b *TaskBuilder) Once(at time.Time) *TaskBuilder {
	b.task.Schedule = Schedule{Type: ScheduleOnce, At: at.Format(time.RFC3339)}
	return b
}

// Timeout sets the task timeout
func (b *TaskBuilder) Timeout(timeout time.Duration) *TaskBuilder {
	b.task.Timeout = timeout
	return b
}

// Handler sets the task handler
func (b *TaskBuilder) Handler(handler TaskHandler) *TaskBuilder {
	b.task.Handler = handler
	return b
}

// Disabled creates the task in disabled state
func (b *TaskBuilder) Disabled() *TaskBuilder {
	b.task.Enabled = false
	return b
}

// Build returns the constructed task
func (b *TaskBuilder) Build() *Task {
	return b.task
}
