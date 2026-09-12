package subscription

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"
)

// SchedulePolicy controls automatic refreshes for one subscription. The
// scheduler intentionally keeps this separate from uif.json; callers can build
// it from their configuration and pass the already validated snapshot.
type SchedulePolicy struct {
	UpdateEnabled     bool          `json:"update_enabled"`
	UpdateInterval    time.Duration `json:"-"`
	UpdateIntervalSec int64         `json:"update_interval_sec"`
	StartupUpdate     bool          `json:"startup_update"`
	StartupJitter     time.Duration `json:"-"`
	StartupJitterSec  int64         `json:"startup_jitter_sec,omitempty"`
}

func (p SchedulePolicy) interval() time.Duration {
	if p.UpdateInterval > 0 {
		return p.UpdateInterval
	}
	if p.UpdateIntervalSec > 0 {
		return time.Duration(p.UpdateIntervalSec) * time.Second
	}
	return 0
}

func (p SchedulePolicy) jitter() time.Duration {
	if p.StartupJitter > 0 {
		return p.StartupJitter
	}
	if p.StartupJitterSec > 0 {
		return time.Duration(p.StartupJitterSec) * time.Second
	}
	return 0
}

// SubscriptionSpec is the complete input to a scheduled execution. Snapshot
// is passed through unchanged so the executor can parse/fetch and publish it
// according to the snapshot policy it received.
type SubscriptionSpec struct {
	ID       string         `json:"id"`
	Source   string         `json:"source,omitempty"`
	Snapshot Snapshot       `json:"snapshot"`
	Policy   SchedulePolicy `json:"policy"`
}

// SchedulerExecutor performs one refresh. It must honor ctx cancellation.
type SchedulerExecutor func(context.Context, SubscriptionSpec) error

// SchedulerOption configures a Scheduler.
type SchedulerOption func(*Scheduler)

// WithExecutor sets the executor when using NewScheduler(nil, options...).
func WithExecutor(fn SchedulerExecutor) SchedulerOption {
	return func(s *Scheduler) { s.executor = fn }
}

// WithJobManager makes scheduled work visible through an existing Manager.
// The manager is not closed by Scheduler.Stop.
func WithJobManager(manager *Manager) SchedulerOption {
	return func(s *Scheduler) {
		if manager != nil {
			s.manager = manager
		}
	}
}

// WithStartupJitter supplies a maximum deterministic startup spread for
// subscriptions that do not specify their own StartupJitter.
func WithStartupJitter(max time.Duration) SchedulerOption {
	return func(s *Scheduler) {
		if max > 0 {
			s.startupJitter = max
		}
	}
}

// ScheduleTask is the scheduler-owned status record. Manager jobs remain the
// source of detailed result/error information and are available by JobID.
type ScheduleTask struct {
	SubscriptionID      string    `json:"subscription_id"`
	State               JobState  `json:"state"`
	JobID               string    `json:"job_id,omitempty"`
	LastStartedAt       time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt      time.Time `json:"last_finished_at,omitempty"`
	NextRunAt           time.Time `json:"next_run_at,omitempty"`
	Error               string    `json:"error,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures,omitempty"`
}

// Scheduler manages cancellable periodic subscription jobs.
type Scheduler struct {
	mu            sync.Mutex
	executor      SchedulerExecutor
	manager       *Manager
	startupJitter time.Duration
	specs         map[string]SubscriptionSpec
	tasks         map[string]*ScheduleTask
	loops         map[string]context.CancelFunc
	loopWG        sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	running       bool
}

// NewScheduler creates a reusable scheduler. A nil executor is accepted so it
// can be installed later with WithExecutor; starting without one is an error.
func NewScheduler(executor SchedulerExecutor, options ...SchedulerOption) *Scheduler {
	s := &Scheduler{
		executor: executor,
		manager:  NewManager(),
		specs:    make(map[string]SubscriptionSpec),
		tasks:    make(map[string]*ScheduleTask),
		loops:    make(map[string]context.CancelFunc),
	}
	for _, option := range options {
		if option != nil {
			option(s)
		}
	}
	return s
}

// Start starts all currently loaded enabled subscriptions. It is safe to call
// repeatedly; a second call is a no-op.
func (s *Scheduler) Start() error {
	return s.StartContext(context.Background())
}

// StartContext is like Start but binds all loops to ctx as well as Stop.
func (s *Scheduler) StartContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	if s.executor == nil {
		return errors.New("subscription scheduler executor is nil")
	}
	s.ctx, s.cancel = context.WithCancel(ctx)
	s.running = true
	for id := range s.specs {
		s.startLoopLocked(id)
	}
	return nil
}

// Stop cancels every timer and in-flight job. It does not discard specs or
// terminal statuses, so Start can be called again.
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	cancel := s.cancel
	s.cancel = nil
	for id, stop := range s.loops {
		stop()
		delete(s.loops, id)
	}
	if cancel != nil {
		cancel()
	}
	s.mu.Unlock()
	s.loopWG.Wait()
	return nil
}

// Reload atomically replaces the set of scheduled subscriptions. Existing
// loops are cancelled before new loops are installed, preventing stale specs
// from running after a configuration update.
func (s *Scheduler) Reload(specs []SubscriptionSpec) error {
	incoming := make(map[string]SubscriptionSpec, len(specs))
	for _, spec := range specs {
		if spec.ID == "" {
			return errors.New("subscription scheduler requires a subscription id")
		}
		if _, exists := incoming[spec.ID]; exists {
			return fmt.Errorf("duplicate subscription id %q", spec.ID)
		}
		incoming[spec.ID] = spec
	}

	s.mu.Lock()
	wasRunning := s.running
	for id, stop := range s.loops {
		stop()
		delete(s.loops, id)
	}
	s.specs = incoming
	for id := range incoming {
		if _, exists := s.tasks[id]; !exists {
			s.tasks[id] = &ScheduleTask{SubscriptionID: id, State: Queued}
		}
	}
	for id, task := range s.tasks {
		if _, exists := incoming[id]; !exists {
			task.State = Cancelled
			task.NextRunAt = time.Time{}
			task.Error = "subscription removed from schedule"
		}
	}
	if wasRunning {
		for id := range incoming {
			s.startLoopLocked(id)
		}
	}
	s.mu.Unlock()
	return nil
}

// RunNow queues one execution immediately. The same per-subscription guard as
// periodic work is used, so concurrent manual and periodic runs cannot enter
// the executor twice.
func (s *Scheduler) RunNow(id string) *Job {
	s.mu.Lock()
	spec, ok := s.specs[id]
	if !ok || !s.running || s.executor == nil {
		s.mu.Unlock()
		return nil
	}
	job := s.startJobLocked(id, spec, s.ctx)
	s.mu.Unlock()
	return job
}

// Cancel cancels the currently running job for a subscription.
func (s *Scheduler) Cancel(id string) bool {
	s.mu.Lock()
	task := s.tasks[id]
	var jobID string
	if task != nil {
		jobID = task.JobID
	}
	s.mu.Unlock()
	return jobID != "" && s.manager.Cancel(jobID)
}

// GetTask returns a copy of a scheduler status record.
func (s *Scheduler) GetTask(id string) *ScheduleTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyScheduleTask(s.tasks[id])
}

// Tasks returns a stable, sorted snapshot of all current and removed task
// records. Removed records remain until the next Reload, which preserves the
// terminal state useful to API callers during a reload transition.
func (s *Scheduler) Tasks() []*ScheduleTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.tasks))
	for id := range s.tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]*ScheduleTask, 0, len(ids))
	for _, id := range ids {
		result = append(result, copyScheduleTask(s.tasks[id]))
	}
	return result
}

// Job returns the detailed Manager record for a scheduled task.
func (s *Scheduler) Job(id string) *Job {
	task := s.GetTask(id)
	if task == nil || task.JobID == "" {
		return nil
	}
	return s.manager.Get(task.JobID)
}

func copyScheduleTask(task *ScheduleTask) *ScheduleTask {
	if task == nil {
		return nil
	}
	copy := *task
	return &copy
}

func (s *Scheduler) startLoopLocked(id string) {
	if _, exists := s.loops[id]; exists {
		return
	}
	spec, exists := s.specs[id]
	if !exists || !spec.Policy.UpdateEnabled || spec.Policy.interval() <= 0 || !s.running {
		return
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.loops[id] = cancel
	s.loopWG.Add(1)
	go s.loop(ctx, spec)
}

func (s *Scheduler) loop(ctx context.Context, spec SubscriptionSpec) {
	defer s.loopWG.Done()
	interval := spec.Policy.interval()
	delay := s.initialDelay(spec)
	if !spec.Policy.StartupUpdate {
		delay += interval
	}
	if !waitContext(ctx, delay) {
		return
	}
	for {
		s.mu.Lock()
		if s.running && ctx.Err() == nil {
			s.startJobLocked(spec.ID, spec, ctx)
		}
		s.mu.Unlock()
		if !waitContext(ctx, interval) {
			return
		}
	}
}

func (s *Scheduler) initialDelay(spec SubscriptionSpec) time.Duration {
	max := spec.Policy.jitter()
	if max <= 0 {
		max = s.startupJitter
	}
	if max <= 0 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(spec.ID))
	return time.Duration(uint64(h.Sum32()) % uint64(max))
}

func waitContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Scheduler) startJobLocked(id string, spec SubscriptionSpec, parent context.Context) *Job {
	if parent == nil {
		parent = context.Background()
	}
	job := s.manager.Start(parent, id, "refresh", func(ctx context.Context) error {
		return s.executor(ctx, spec)
	})
	task := s.tasks[id]
	if task == nil {
		task = &ScheduleTask{SubscriptionID: id}
		s.tasks[id] = task
	}
	if task.JobID != job.ID || task.State == Success || task.State == Failed || task.State == Cancelled {
		task.JobID = job.ID
		task.State = job.State
		task.LastStartedAt = time.Time{}
		task.LastFinishedAt = time.Time{}
		task.Error = ""
		task.NextRunAt = time.Now().Add(spec.Policy.interval())
	}
	go s.observe(id, job.ID)
	return job
}

func (s *Scheduler) observe(id, jobID string) {
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		job := s.manager.Get(jobID)
		if job == nil {
			return
		}
		s.mu.Lock()
		task := s.tasks[id]
		if task != nil && task.JobID == jobID {
			task.State = job.State
			task.LastStartedAt = job.StartedAt
			task.LastFinishedAt = job.FinishedAt
			task.Error = job.Error
			if job.State == Success {
				task.ConsecutiveFailures = 0
			} else if job.State == Failed {
				task.ConsecutiveFailures++
			}
		}
		s.mu.Unlock()
		if job.State == Success || job.State == Failed || job.State == Cancelled {
			return
		}
	}
}
