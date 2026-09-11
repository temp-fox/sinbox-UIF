package subscription

import (
	"context"
	"sync"
	"time"
)

type JobState string

const (
	Queued    JobState = "queued"
	Running   JobState = "running"
	Success   JobState = "success"
	Failed    JobState = "failed"
	Cancelled JobState = "cancelled"
)

type Job struct {
	ID             string    `json:"job_id"`
	Type           string    `json:"type"`
	State          JobState  `json:"state"`
	SubscriptionID string    `json:"subscription_id"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
	Error          string    `json:"error,omitempty"`
}

type Manager struct {
	mu     sync.Mutex
	jobs   map[string]*Job
	cancel map[string]context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{jobs: make(map[string]*Job), cancel: make(map[string]context.CancelFunc)}
}

func (m *Manager) Start(ctx context.Context, id, kind string, fn func(context.Context) error) *Job {
	m.mu.Lock()
	jobID := time.Now().Format("20060102150405.000000000")
	jobCtx, cancel := context.WithCancel(ctx)
	job := &Job{ID: jobID, Type: kind, SubscriptionID: id, State: Queued}
	m.jobs[jobID] = job
	m.cancel[jobID] = cancel
	m.mu.Unlock()
	go m.run(job, jobCtx, cancel, fn)
	return job
}

func (m *Manager) run(job *Job, ctx context.Context, cancel context.CancelFunc, fn func(context.Context) error) {
	defer cancel()
	m.mu.Lock()
	job.State = Running
	job.StartedAt = time.Now()
	m.mu.Unlock()
	err := fn(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	job.FinishedAt = time.Now()
	if ctx.Err() != nil {
		job.State = Cancelled
		job.Error = ctx.Err().Error()
	} else if err != nil {
		job.State = Failed
		job.Error = err.Error()
	} else {
		job.State = Success
	}
	delete(m.cancel, job.ID)
}

func (m *Manager) Get(id string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job := m.jobs[id]; job != nil {
		copy := *job
		return &copy
	}
	return nil
}

func (m *Manager) Cancel(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel := m.cancel[id]; cancel != nil {
		cancel()
		return true
	}
	return false
}
