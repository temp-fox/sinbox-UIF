package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/uif/uifd/subscription"
)

func TestSubscriptionTaskAPIListsGetsAndCancelsSchedulerTask(t *testing.T) {
	oldScheduler := subscriptionScheduler
	defer func() { subscriptionScheduler = oldScheduler }()

	started := make(chan struct{})
	scheduler := subscription.NewScheduler(func(ctx context.Context, _ subscription.SubscriptionSpec) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	if err := scheduler.Reload([]subscription.SubscriptionSpec{{ID: "api-task", Policy: subscription.SchedulePolicy{UpdateEnabled: true}}}); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Start(); err != nil {
		t.Fatal(err)
	}
	defer scheduler.Stop()
	subscriptionScheduler = scheduler
	job := scheduler.RunNow("api-task")
	if job == nil {
		t.Fatal("RunNow returned nil")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("scheduler job did not start")
	}

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/tasks", nil)
	resp := httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", resp.Code, resp.Body.String())
	}
	var tasks []*subscription.ScheduleTask
	if err := json.Unmarshal(resp.Body.Bytes(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].SubscriptionID != "api-task" {
		t.Fatalf("unexpected task list: %#v", tasks)
	}

	req = httptest.NewRequest(http.MethodGet, "/subscriptions/task?id=api-task", nil)
	resp = httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "api-task") {
		t.Fatalf("get response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/subscriptions/task/api-task/cancel", nil)
	resp = httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"cancelled":true`) {
		t.Fatalf("cancel response = %d %s", resp.Code, resp.Body.String())
	}
}

func TestSubscriptionTaskAPIListsAndCancelsManualManagerJob(t *testing.T) {
	oldScheduler := subscriptionScheduler
	oldJobs := subscriptionJobs
	defer func() {
		subscriptionScheduler = oldScheduler
		subscriptionJobs = oldJobs
	}()

	started := make(chan struct{})
	finished := make(chan struct{})
	subscriptionScheduler = subscription.NewScheduler(nil)
	subscriptionJobs = subscription.NewManager()
	job := subscriptionJobs.Start(context.Background(), "manual-task", "refresh", func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(finished)
		return ctx.Err()
	})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("manual job did not start")
	}

	req := httptest.NewRequest(http.MethodGet, "/subscriptions/tasks", nil)
	resp := httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "manual-task") {
		t.Fatalf("list response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/subscriptions/task?id="+job.ID, nil)
	resp = httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), job.ID) {
		t.Fatalf("get response = %d %s", resp.Code, resp.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/subscriptions/task/"+job.ID+"/cancel", nil)
	resp = httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), `"cancelled":true`) {
		t.Fatalf("cancel response = %d %s", resp.Code, resp.Body.String())
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("manual job did not observe cancellation")
	}
}

func TestSubscriptionTaskAPIValidatesMethodsAndIDs(t *testing.T) {
	oldScheduler := subscriptionScheduler
	defer func() { subscriptionScheduler = oldScheduler }()
	subscriptionScheduler = subscription.NewScheduler(nil)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/tasks", nil)
	resp := httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method status = %d, want 405", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/subscriptions/task", nil)
	resp = httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("missing id status = %d, want 400", resp.Code)
	}
}
