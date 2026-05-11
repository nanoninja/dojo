// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/handler"
)

// fakeDBPinger lets tests control DB ping behavior.
type fakeDBPinger struct {
	pingFn func(ctx context.Context) error
}

func (f fakeDBPinger) PingContext(ctx context.Context) error {
	if f.pingFn != nil {
		return f.pingFn(ctx)
	}
	return nil
}

// fakeCachePinger lets tests control cache ping behavior.
type fakeCachePinger struct {
	pingFn func(ctx context.Context) error
}

func (f fakeCachePinger) Ping(ctx context.Context) error {
	if f.pingFn != nil {
		return f.pingFn(ctx)
	}
	return nil
}

func TestHealthHandler_Health_OK(t *testing.T) {
	h := handler.NewHealthHandler(
		"v1.0.0",
		"test",
		fakeDBPinger{},
		fakeCachePinger{},
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	serve(h.Health, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Equal(t, "ok", body["status"].(string))
	assert.Equal(t, true, body["db"].(bool))
	assert.Equal(t, true, body["cache"].(bool))
}

func TestHealthHandler_Health_Degraded_WhenDBFails(t *testing.T) {
	h := handler.NewHealthHandler(
		"v1.0.0",
		"test",
		fakeDBPinger{
			pingFn: func(_ context.Context) error {
				return errors.New("db down")
			},
		},
		fakeCachePinger{},
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	serve(h.Health, w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Equal(t, "degraded", body["status"].(string))
	assert.Equal(t, false, body["db"].(bool))
	assert.Equal(t, true, body["cache"].(bool))
}

func TestHealthHandler_Live_OK(t *testing.T) {
	h := handler.NewHealthHandler(
		"v1.0.0",
		"test",
		fakeDBPinger{},
		fakeCachePinger{},
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/livez", nil)

	serve(h.Live, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Equal(t, "ok", body["status"].(string))
	assert.Equal(t, "v1.0.0", body["version"].(string))
	assert.Equal(t, "test", body["env"].(string))
	assert.NotEqual(t, "", body["uptime"].(string), "uptime should not be empty")
}

func TestHealthHandler_Ready_OK(t *testing.T) {
	h := handler.NewHealthHandler(
		"v1.0.0",
		"test",
		fakeDBPinger{},
		fakeCachePinger{},
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/readyz", nil)

	serve(h.Ready, w, r)

	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Equal(t, "ready", body["status"].(string))

	checks := body["checks"].(map[string]any)
	assert.Equal(t, true, checks["db"].(bool))
	assert.Equal(t, true, checks["cache"].(bool))
}

func TestHealthHandler_Ready_NotReady_WhenCacheFails(t *testing.T) {
	h := handler.NewHealthHandler(
		"v1.0.0",
		"test",
		fakeDBPinger{},
		fakeCachePinger{
			pingFn: func(_ context.Context) error {
				return errors.New("cache down")
			},
		},
	)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/readyz", nil)

	serve(h.Ready, w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body map[string]any
	decodeJSON(t, w, &body)

	assert.Equal(t, "not_ready", body["status"].(string))

	checks := body["checks"].(map[string]any)
	assert.Equal(t, true, checks["db"].(bool))
	assert.Equal(t, false, checks["cache"].(bool))
}

func TestHealthHandler_Ready_Timeout(t *testing.T) {
	h := handler.NewHealthHandler(
		"v1.0.0",
		"test",
		fakeDBPinger{
			pingFn: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		},
		fakeCachePinger{
			pingFn: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
		},
	)

	start := time.Now()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/readyz", nil)

	serve(h.Ready, w, r)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	// The handler timeout is 300ms, so this request should not run for seconds.
	assert.True(t, time.Since(start) <= 2*time.Second, "Ready() took too long")
}
