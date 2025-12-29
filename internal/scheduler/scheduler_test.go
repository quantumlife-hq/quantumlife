package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timezone != "Local" {
		t.Errorf("Timezone = %v, want Local", cfg.Timezone)
	}
}

func TestNewScheduler(t *testing.T) {
	t.Run("with valid timezone", func(t *testing.T) {
		cfg := Config{Timezone: "America/New_York"}
		s, err := NewScheduler(cfg)

		if err != nil {
			t.Fatalf("NewScheduler failed: %v", err)
		}
		if s == nil {
			t.Fatal("scheduler is nil")
		}
		if s.tasks == nil {
			t.Error("tasks map is nil")
		}
		if s.running == nil {
			t.Error("running map is nil")
		}
	})

	t.Run("with invalid timezone uses local", func(t *testing.T) {
		cfg := Config{Timezone: "Invalid/Timezone"}
		s, err := NewScheduler(cfg)

		if err != nil {
			t.Fatalf("NewScheduler failed: %v", err)
		}
		if s == nil {
			t.Fatal("scheduler is nil")
		}
		// Should fall back to local timezone
	})
}

func TestScheduler_Register(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	t.Run("valid task", func(t *testing.T) {
		task := &Task{
			ID:       "test-1",
			Name:     "Test Task",
			Handler:  func(ctx context.Context) error { return nil },
			Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
		}

		err := s.Register(task)
		if err != nil {
			t.Errorf("Register failed: %v", err)
		}

		// Check task was registered
		if _, ok := s.tasks["test-1"]; !ok {
			t.Error("task not found in scheduler")
		}

		// Check defaults were set
		if task.Timeout == 0 {
			t.Error("default timeout not set")
		}
		if !task.Enabled {
			t.Error("task should be enabled by default")
		}
		if task.NextRun == nil {
			t.Error("NextRun not calculated")
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		task := &Task{
			Handler: func(ctx context.Context) error { return nil },
		}

		err := s.Register(task)
		if err == nil {
			t.Error("expected error for empty ID")
		}
	})

	t.Run("nil handler", func(t *testing.T) {
		task := &Task{
			ID: "test-2",
		}

		err := s.Register(task)
		if err == nil {
			t.Error("expected error for nil handler")
		}
	})

	t.Run("with custom timeout", func(t *testing.T) {
		task := &Task{
			ID:       "test-3",
			Handler:  func(ctx context.Context) error { return nil },
			Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
			Timeout:  10 * time.Minute,
		}

		err := s.Register(task)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		if task.Timeout != 10*time.Minute {
			t.Error("custom timeout overwritten")
		}
	})
}

func TestScheduler_Unregister(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	task := &Task{
		ID:       "test-1",
		Handler:  func(ctx context.Context) error { return nil },
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
	}
	s.Register(task)

	err := s.Unregister("test-1")
	if err != nil {
		t.Errorf("Unregister failed: %v", err)
	}

	if _, ok := s.tasks["test-1"]; ok {
		t.Error("task should be removed")
	}
}

func TestScheduler_Unregister_Nonexistent(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	err := s.Unregister("nonexistent")
	if err != nil {
		t.Error("should not error for nonexistent task")
	}
}

func TestScheduler_Enable(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	task := &Task{
		ID:       "test-1",
		Handler:  func(ctx context.Context) error { return nil },
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
		Enabled:  false,
	}
	s.Register(task)
	task.Enabled = false // Reset after register

	err := s.Enable("test-1")
	if err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	if !task.Enabled {
		t.Error("task should be enabled")
	}
}

func TestScheduler_Enable_NotFound(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	err := s.Enable("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_Disable(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	task := &Task{
		ID:       "test-1",
		Handler:  func(ctx context.Context) error { return nil },
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
	}
	s.Register(task)

	err := s.Disable("test-1")
	if err != nil {
		t.Errorf("Disable failed: %v", err)
	}

	if task.Enabled {
		t.Error("task should be disabled")
	}
}

func TestScheduler_Disable_NotFound(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	err := s.Disable("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_Start(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	err := s.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !s.started {
		t.Error("scheduler should be started")
	}

	s.Stop()
}

func TestScheduler_Start_AlreadyStarted(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())
	s.Start()
	defer s.Stop()

	err := s.Start()
	if err == nil {
		t.Error("expected error when already started")
	}
}

func TestScheduler_Stop(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())
	s.Start()

	err := s.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	if s.started {
		t.Error("scheduler should be stopped")
	}
}

func TestScheduler_Stop_NotStarted(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	err := s.Stop()
	if err != nil {
		t.Errorf("Stop should not error when not started: %v", err)
	}
}

func TestScheduler_GetTask(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	task := &Task{
		ID:       "test-1",
		Handler:  func(ctx context.Context) error { return nil },
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
	}
	s.Register(task)

	t.Run("found", func(t *testing.T) {
		got, ok := s.GetTask("test-1")
		if !ok {
			t.Error("task should be found")
		}
		if got.ID != "test-1" {
			t.Error("wrong task returned")
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, ok := s.GetTask("nonexistent")
		if ok {
			t.Error("should not find nonexistent task")
		}
	})
}

func TestScheduler_ListTasks(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	handler := func(ctx context.Context) error { return nil }

	s.Register(&Task{ID: "1", Handler: handler, Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute}})
	s.Register(&Task{ID: "2", Handler: handler, Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute}})

	tasks := s.ListTasks()
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestScheduler_GetStats(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	// Use long interval to prevent task from running during test
	handler := func(ctx context.Context) error { return nil }
	s.Register(&Task{ID: "1", Handler: handler, Schedule: Schedule{Type: ScheduleInterval, Interval: time.Hour}})
	s.Register(&Task{ID: "2", Handler: handler, Schedule: Schedule{Type: ScheduleInterval, Interval: time.Hour}})

	// Get stats before starting (no deadlock risk)
	stats := s.GetStats()

	if stats.Started {
		t.Error("Started should be false before Start")
	}
	if stats.TotalTasks != 2 {
		t.Errorf("TotalTasks = %d, want 2", stats.TotalTasks)
	}
	if stats.EnabledTasks != 2 {
		t.Errorf("EnabledTasks = %d, want 2", stats.EnabledTasks)
	}
}

func TestScheduler_RunNow(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	executed := make(chan bool, 1)
	task := &Task{
		ID: "test-1",
		Handler: func(ctx context.Context) error {
			executed <- true
			return nil
		},
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Hour},
	}
	s.Register(task)

	err := s.RunNow("test-1")
	if err != nil {
		t.Errorf("RunNow failed: %v", err)
	}

	select {
	case <-executed:
		// Task executed
	case <-time.After(time.Second):
		t.Error("task not executed within timeout")
	}
}

func TestScheduler_RunNow_NotFound(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	err := s.RunNow("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestScheduler_ExecuteTask_Error(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	task := &Task{
		ID: "test-1",
		Handler: func(ctx context.Context) error {
			return errors.New("test error")
		},
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
		Timeout:  time.Second,
	}
	s.Register(task)

	s.executeTask(context.Background(), task)

	if task.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", task.ErrorCount)
	}
	if task.LastError != "test error" {
		t.Errorf("LastError = %v, want 'test error'", task.LastError)
	}
	if task.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", task.RunCount)
	}
}

func TestScheduler_ExecuteTask_Success(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	task := &Task{
		ID: "test-1",
		Handler: func(ctx context.Context) error {
			return nil
		},
		Schedule: Schedule{Type: ScheduleInterval, Interval: time.Minute},
		Timeout:  time.Second,
	}
	s.Register(task)

	s.executeTask(context.Background(), task)

	if task.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0", task.ErrorCount)
	}
	if task.LastError != "" {
		t.Error("LastError should be empty on success")
	}
	if task.LastRun == nil {
		t.Error("LastRun should be set")
	}
}

func TestScheduler_CalculateNextRun(t *testing.T) {
	s, _ := NewScheduler(DefaultConfig())

	t.Run("interval", func(t *testing.T) {
		schedule := Schedule{Type: ScheduleInterval, Interval: 10 * time.Minute}
		next := s.calculateNextRun(schedule)

		expected := time.Now().Add(10 * time.Minute)
		if next.Before(expected.Add(-time.Second)) || next.After(expected.Add(time.Second)) {
			t.Errorf("next = %v, want ~%v", next, expected)
		}
	})

	t.Run("daily", func(t *testing.T) {
		schedule := Schedule{Type: ScheduleDaily, At: "14:30"}
		next := s.calculateNextRun(schedule)

		if next.Hour() != 14 || next.Minute() != 30 {
			t.Errorf("next time = %02d:%02d, want 14:30", next.Hour(), next.Minute())
		}
	})

	t.Run("weekly", func(t *testing.T) {
		schedule := Schedule{
			Type: ScheduleWeekly,
			At:   "09:00",
			Days: []time.Weekday{time.Monday, time.Wednesday, time.Friday},
		}
		next := s.calculateNextRun(schedule)

		if next.Hour() != 9 || next.Minute() != 0 {
			t.Errorf("next time = %02d:%02d, want 09:00", next.Hour(), next.Minute())
		}

		validDay := false
		for _, day := range schedule.Days {
			if next.Weekday() == day {
				validDay = true
				break
			}
		}
		if !validDay {
			t.Errorf("next day = %v, not in schedule", next.Weekday())
		}
	})

	t.Run("once", func(t *testing.T) {
		target := time.Now().Add(time.Hour)
		schedule := Schedule{Type: ScheduleOnce, At: target.Format(time.RFC3339)}
		next := s.calculateNextRun(schedule)

		if next.Sub(target) > time.Second {
			t.Errorf("next = %v, want %v", next, target)
		}
	})

	t.Run("once invalid", func(t *testing.T) {
		schedule := Schedule{Type: ScheduleOnce, At: "invalid"}
		next := s.calculateNextRun(schedule)

		// Should return soon
		if next.After(time.Now().Add(2 * time.Minute)) {
			t.Error("should return near future for invalid once")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		schedule := Schedule{Type: ScheduleType("unknown")}
		next := s.calculateNextRun(schedule)

		// Should default to 1 hour
		expected := time.Now().Add(time.Hour)
		if next.Before(expected.Add(-time.Second)) || next.After(expected.Add(time.Second)) {
			t.Errorf("next = %v, want ~%v", next, expected)
		}
	})
}

func TestScheduler_TaskExecution_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s, _ := NewScheduler(DefaultConfig())

	var count int32
	done := make(chan struct{})
	task := &Task{
		ID: "test-1",
		Handler: func(ctx context.Context) error {
			if atomic.AddInt32(&count, 1) >= 2 {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			return nil
		},
		Schedule: Schedule{Type: ScheduleInterval, Interval: 10 * time.Millisecond},
		Timeout:  time.Second,
	}
	s.Register(task)
	s.Start()

	// Wait for at least 2 executions or timeout
	select {
	case <-done:
		// Success
	case <-time.After(500 * time.Millisecond):
		// Timeout - check if we got at least one
	}

	s.Stop()

	if atomic.LoadInt32(&count) < 1 {
		t.Errorf("count = %d, expected at least 1 execution", count)
	}
}

// Task builder tests

func TestIntervalTask(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }
	task := IntervalTask("test", "Test Task", time.Minute, handler)

	if task.ID != "test" {
		t.Error("ID not set correctly")
	}
	if task.Name != "Test Task" {
		t.Error("Name not set correctly")
	}
	if task.Schedule.Type != ScheduleInterval {
		t.Error("Schedule type should be interval")
	}
	if task.Schedule.Interval != time.Minute {
		t.Error("Interval not set correctly")
	}
}

func TestDailyTask(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }
	task := DailyTask("test", "Test Task", "08:00", handler)

	if task.ID != "test" {
		t.Error("ID not set correctly")
	}
	if task.Schedule.Type != ScheduleDaily {
		t.Error("Schedule type should be daily")
	}
	if task.Schedule.At != "08:00" {
		t.Error("At not set correctly")
	}
}

func TestWeeklyTask(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }
	days := []time.Weekday{time.Monday, time.Friday}
	task := WeeklyTask("test", "Test Task", "09:00", days, handler)

	if task.Schedule.Type != ScheduleWeekly {
		t.Error("Schedule type should be weekly")
	}
	if len(task.Schedule.Days) != 2 {
		t.Error("Days not set correctly")
	}
}

func TestOnceTask(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }
	at := time.Now().Add(time.Hour)
	task := OnceTask("test", "Test Task", at, handler)

	if task.Schedule.Type != ScheduleOnce {
		t.Error("Schedule type should be once")
	}
	if task.Schedule.At != at.Format(time.RFC3339) {
		t.Error("At not set correctly")
	}
}

// TaskBuilder tests

func TestNewTask(t *testing.T) {
	builder := NewTask("test-id")

	if builder == nil {
		t.Fatal("NewTask returned nil")
	}
	if builder.task.ID != "test-id" {
		t.Error("ID not set correctly")
	}
	if !builder.task.Enabled {
		t.Error("should be enabled by default")
	}
	if builder.task.Timeout != 5*time.Minute {
		t.Error("default timeout not set")
	}
}

func TestTaskBuilder_FluentAPI(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }

	task := NewTask("test").
		Name("Test Task").
		Description("A test task").
		Every(10 * time.Minute).
		Timeout(time.Hour).
		Handler(handler).
		Build()

	if task.ID != "test" {
		t.Error("ID not set correctly")
	}
	if task.Name != "Test Task" {
		t.Error("Name not set correctly")
	}
	if task.Description != "A test task" {
		t.Error("Description not set correctly")
	}
	if task.Schedule.Type != ScheduleInterval {
		t.Error("Schedule type should be interval")
	}
	if task.Schedule.Interval != 10*time.Minute {
		t.Error("Interval not set correctly")
	}
	if task.Timeout != time.Hour {
		t.Error("Timeout not set correctly")
	}
}

func TestTaskBuilder_Daily(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }

	task := NewTask("test").
		Daily("07:30").
		Handler(handler).
		Build()

	if task.Schedule.Type != ScheduleDaily {
		t.Error("Schedule type should be daily")
	}
	if task.Schedule.At != "07:30" {
		t.Error("At not set correctly")
	}
}

func TestTaskBuilder_Weekly(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }

	task := NewTask("test").
		Weekly("09:00", time.Monday, time.Wednesday).
		Handler(handler).
		Build()

	if task.Schedule.Type != ScheduleWeekly {
		t.Error("Schedule type should be weekly")
	}
	if len(task.Schedule.Days) != 2 {
		t.Error("Days not set correctly")
	}
}

func TestTaskBuilder_Once(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }
	at := time.Now().Add(time.Hour)

	task := NewTask("test").
		Once(at).
		Handler(handler).
		Build()

	if task.Schedule.Type != ScheduleOnce {
		t.Error("Schedule type should be once")
	}
}

func TestTaskBuilder_Disabled(t *testing.T) {
	handler := func(ctx context.Context) error { return nil }

	task := NewTask("test").
		Handler(handler).
		Disabled().
		Build()

	if task.Enabled {
		t.Error("task should be disabled")
	}
}

// Struct field tests

func TestTask_Fields(t *testing.T) {
	now := time.Now()
	task := Task{
		ID:          "test-id",
		Name:        "Test Task",
		Description: "Description",
		Schedule:    Schedule{Type: ScheduleDaily, At: "08:00"},
		Enabled:     true,
		LastRun:     &now,
		NextRun:     &now,
		RunCount:    5,
		ErrorCount:  1,
		LastError:   "some error",
		CreatedAt:   now,
		Timeout:     time.Minute,
	}

	if task.ID != "test-id" {
		t.Error("ID not set correctly")
	}
	if task.RunCount != 5 {
		t.Error("RunCount not set correctly")
	}
	if task.ErrorCount != 1 {
		t.Error("ErrorCount not set correctly")
	}
}

func TestSchedule_Fields(t *testing.T) {
	schedule := Schedule{
		Type:     ScheduleWeekly,
		Interval: time.Hour,
		Cron:     "0 8 * * *",
		At:       "08:00",
		Days:     []time.Weekday{time.Monday},
	}

	if schedule.Type != ScheduleWeekly {
		t.Error("Type not set correctly")
	}
	if schedule.Interval != time.Hour {
		t.Error("Interval not set correctly")
	}
	if schedule.Cron != "0 8 * * *" {
		t.Error("Cron not set correctly")
	}
}

func TestScheduleType(t *testing.T) {
	types := []struct {
		st   ScheduleType
		want string
	}{
		{ScheduleInterval, "interval"},
		{ScheduleDaily, "daily"},
		{ScheduleWeekly, "weekly"},
		{ScheduleCron, "cron"},
		{ScheduleOnce, "once"},
	}

	for _, tt := range types {
		if string(tt.st) != tt.want {
			t.Errorf("%v = %v, want %v", tt.st, string(tt.st), tt.want)
		}
	}
}

func TestStats_Fields(t *testing.T) {
	stats := Stats{
		Started:      true,
		TotalTasks:   10,
		EnabledTasks: 8,
		RunningTasks: 5,
		TotalRuns:    100,
		TotalErrors:  3,
		Timezone:     "America/New_York",
	}

	if !stats.Started {
		t.Error("Started not set correctly")
	}
	if stats.TotalTasks != 10 {
		t.Error("TotalTasks not set correctly")
	}
	if stats.TotalRuns != 100 {
		t.Error("TotalRuns not set correctly")
	}
}

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		Timezone: "Europe/London",
	}

	if cfg.Timezone != "Europe/London" {
		t.Error("Timezone not set correctly")
	}
}
