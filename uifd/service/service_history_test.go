package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSubscriptionSnapshotAPIRestoreRejectsTraversal(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/subscriptions/restore", strings.NewReader(url.Values{"id": {"api"}, "history": {"../../outside.json"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	Service(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("traversal status = %d, body=%s", resp.Code, resp.Body.String())
	}
}
