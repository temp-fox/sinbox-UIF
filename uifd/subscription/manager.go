package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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
	ID              string           `json:"job_id"`
	Type            string           `json:"type"`
	State           JobState         `json:"state"`
	SubscriptionID  string           `json:"subscription_id"`
	StartedAt       time.Time        `json:"started_at,omitempty"`
	FinishedAt      time.Time        `json:"finished_at,omitempty"`
	Error           string           `json:"error,omitempty"`
	Result          string           `json:"result,omitempty"`
	ResultBytes     int              `json:"result_bytes,omitempty"`
	ExtraInfo       string           `json:"extra_info,omitempty"`
	ParseSummary    *ParseSummary    `json:"parse_summary,omitempty"`
	SnapshotSummary *SnapshotSummary `json:"snapshot_summary,omitempty"`
	ProbeSummary    *ProbeSummary    `json:"probe_summary,omitempty"`
}

type ParseSummary struct {
	Format  string `json:"format"`
	Nodes   int    `json:"nodes"`
	Skipped int    `json:"skipped"`
}

type ProbeSummary struct {
	Status    string `json:"status"`
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Healthy   int    `json:"healthy"`
	Failed    int    `json:"failed"`
}

// JobFunc 是单个订阅任务的执行回调。回调应监听 ctx.Done，以便停止正在运行的任务。
type JobFunc func(context.Context) error

type ResultFunc func(context.Context) (string, error)

// ManagerOption 配置 Manager。
type ManagerOption func(*Manager)

// WithMaxConcurrent 限制同时运行的回调数量。小于 1 时按 1 处理。
func WithMaxConcurrent(n int) ManagerOption {
	return func(m *Manager) {
		if n < 1 {
			n = 1
		}
		m.limit = n
		m.slots = make(chan struct{}, n)
	}
}

// WithRetention 控制已完成任务状态在内存中的保留时间。非正值表示不自动过期。
func WithRetention(d time.Duration) ManagerOption {
	return func(m *Manager) { m.retention = d }
}

// WithCallback 设置 Start 传入 nil 时使用的默认回调。
func WithCallback(fn JobFunc) ManagerOption {
	return func(m *Manager) { m.callback = fn }
}

type Manager struct {
	mu        sync.Mutex
	jobs      map[string]*Job
	cancel    map[string]context.CancelFunc
	active    map[string]string
	callback  JobFunc
	retention time.Duration
	limit     int
	slots     chan struct{}
	closed    bool
	sequence  uint64
}

// NewManager 创建 Manager。默认最多同时执行 4 个回调，并保留终态 10 分钟。
func NewManager(options ...ManagerOption) *Manager {
	m := &Manager{
		jobs:      make(map[string]*Job),
		cancel:    make(map[string]context.CancelFunc),
		active:    make(map[string]string),
		callback:  func(context.Context) error { return nil },
		retention: 10 * time.Minute,
		limit:     4,
	}
	m.slots = make(chan struct{}, m.limit)
	for _, option := range options {
		if option != nil {
			option(m)
		}
	}
	return m
}

// SetCallback 修改后续任务使用的默认回调。Start 直接传入的回调优先。
func (m *Manager) SetCallback(fn JobFunc) {
	m.mu.Lock()
	m.callback = fn
	m.mu.Unlock()
}

// StartResult runs a callback that can return a result payload and byte count.
func (m *Manager) StartResult(ctx context.Context, id, kind string, fn ResultFunc) *Job {
	if fn == nil {
		return m.Start(ctx, id, kind, nil)
	}
	return m.start(ctx, id, kind, nil, fn)
}

func (m *Manager) Start(ctx context.Context, id, kind string, fn JobFunc) *Job {
	return m.start(ctx, id, kind, fn, nil)
}

func (m *Manager) start(ctx context.Context, id, kind string, fn JobFunc, resultFn ResultFunc) *Job {
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(time.Now())
	if id != "" {
		if existingID := m.active[id]; existingID != "" {
			if existing := m.jobs[existingID]; existing != nil {
				copy := *existing
				return &copy
			}
			delete(m.active, id)
		}
	}
	if m.closed {
		job := &Job{ID: m.nextIDLocked(), Type: kind, SubscriptionID: id, State: Failed, FinishedAt: time.Now(), Error: "manager is closed"}
		m.jobs[job.ID] = job
		return copyJob(job)
	}
	if resultFn == nil {
		if fn == nil {
			fn = m.callback
		}
		if fn == nil {
			fn = func(context.Context) error { return nil }
		}
	}
	jobCtx, cancel := context.WithCancel(ctx)
	job := &Job{ID: m.nextIDLocked(), Type: kind, SubscriptionID: id, State: Queued}
	m.jobs[job.ID] = job
	m.cancel[job.ID] = cancel
	if id != "" {
		m.active[id] = job.ID
	}
	go m.run(job.ID, jobCtx, cancel, fn, resultFn)
	return copyJob(job)
}

func (m *Manager) nextIDLocked() string {
	sequence := atomic.AddUint64(&m.sequence, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), sequence)
}

func (m *Manager) run(jobID string, ctx context.Context, cancel context.CancelFunc, fn JobFunc, resultFn ResultFunc) {
	defer cancel()
	select {
	case m.slots <- struct{}{}:
	case <-ctx.Done():
		m.finish(jobID, Cancelled, ctx.Err())
		return
	}
	defer func() { <-m.slots }()

	m.mu.Lock()
	job := m.jobs[jobID]
	if job == nil {
		m.mu.Unlock()
		return
	}
	job.State = Running
	job.StartedAt = time.Now()
	m.mu.Unlock()

	var err error
	var result string
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				err = fmt.Errorf("subscription job panic: %v", recovered)
			}
		}()
		if resultFn != nil {
			result, err = resultFn(ctx)
		} else {
			err = fn(ctx)
		}
	}()
	if resultFn != nil {
		m.mu.Lock()
		if job := m.jobs[jobID]; job != nil {
			job.Result = result
			job.ResultBytes = len([]byte(result))
			var envelope struct {
				ExtraInfo       string           `json:"extra_info"`
				ParseSummary    *ParseSummary    `json:"parse_summary"`
				SnapshotSummary *SnapshotSummary `json:"snapshot_summary"`
				ProbeSummary    *ProbeSummary    `json:"probe_summary"`
			}
			if json.Unmarshal([]byte(result), &envelope) == nil {
				job.ExtraInfo = envelope.ExtraInfo
				job.ParseSummary = envelope.ParseSummary
				job.SnapshotSummary = envelope.SnapshotSummary
				job.ProbeSummary = envelope.ProbeSummary
			}
		}
		m.mu.Unlock()
	}
	if ctx.Err() != nil {
		m.finish(jobID, Cancelled, ctx.Err())
	} else if err != nil {
		m.finish(jobID, Failed, err)
	} else {
		m.finish(jobID, Success, nil)
	}
}

func (m *Manager) finish(jobID string, state JobState, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job := m.jobs[jobID]
	if job == nil {
		return
	}
	job.State = state
	job.FinishedAt = time.Now()
	if err != nil {
		job.Error = err.Error()
	} else {
		job.Error = ""
	}
	delete(m.cancel, jobID)
	if job.SubscriptionID != "" && m.active[job.SubscriptionID] == jobID {
		delete(m.active, job.SubscriptionID)
	}
}

func copyJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copy := *job
	return &copy
}

// Get 返回任务持久化状态的快照。
func (m *Manager) Get(id string) *Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupLocked(time.Now())
	return copyJob(m.jobs[id])
}

// Cancel 请求取消 queued/running 任务。
func (m *Manager) Cancel(id string) bool {
	m.mu.Lock()
	cancel := m.cancel[id]
	m.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// Cleanup 清理超过保留时间的终态任务，并返回清理数量。
func (m *Manager) Cleanup() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cleanupLocked(time.Now())
}

func (m *Manager) cleanupLocked(now time.Time) int {
	if m.retention <= 0 {
		return 0
	}
	removed := 0
	for id, job := range m.jobs {
		if job.FinishedAt.IsZero() || now.Sub(job.FinishedAt) < m.retention {
			continue
		}
		delete(m.jobs, id)
		delete(m.cancel, id)
		if job.SubscriptionID != "" && m.active[job.SubscriptionID] == id {
			delete(m.active, job.SubscriptionID)
		}
		removed++
	}
	return removed
}

// Close 取消所有 queued/running 任务，并拒绝后续提交。
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	cancels := make([]context.CancelFunc, 0, len(m.cancel))
	for _, cancel := range m.cancel {
		cancels = append(cancels, cancel)
	}
	m.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

// Limit 返回同时运行回调的上限。
func (m *Manager) Limit() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.limit
}
