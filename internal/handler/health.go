// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/nanoninja/dojo/internal/httputil"
)

// Pinger is implemented by any service that can report its availability.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// RedisPinger is implemented by the Redis client.
type RedisPinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler handles the health check endpoint.
type HealthHandler struct {
	version   string
	env       string
	db        Pinger
	cache     RedisPinger
	startedAt time.Time
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(version, env string, db Pinger, cache RedisPinger) *HealthHandler {
	return &HealthHandler{
		version:   version,
		env:       env,
		db:        db,
		cache:     cache,
		startedAt: time.Now(),
	}
}

// HealthResponse is the JSON body returned by the health check endpoint.
type HealthResponse struct {
	Status  string `json:"status"  example:"ok"`
	Version string `json:"version" example:"v1.0.0"`
	Env     string `json:"env"     example:"dev"`
	DB      bool   `json:"db"      example:"true"`
	Cache   bool   `json:"cache"   example:"true"`
}

// LiveResponse is the JSON body returned by the liveness endpoint.
type LiveResponse struct {
	Status  string `json:"status"  example:"ok"`
	Version string `json:"version" example:"v1.0.0"`
	Env     string `json:"env"     example:"dev"`
	Uptime  string `json:"uptime"  example:"1m23s"`
}

// ReadinessChecks contains dependency status used by the readiness endpoint.
type ReadinessChecks struct {
	DB    bool `json:"db"    example:"true"`
	Cache bool `json:"cache" example:"true"`
}

// ReadyResponse is the JSON body returned by the readiness endpoint.
type ReadyResponse struct {
	Status  string          `json:"status"  example:"ready"`
	Version string          `json:"version" example:"v1.0.0"`
	Env     string          `json:"env"     example:"dev"`
	Checks  ReadinessChecks `json:"checks"`
}

// Health handles GET /health
//
// @Summary  Check the health of the API and its dependencies
// @Tags     system
// @Produce  json
// @Success  200  {object} HealthResponse "all services healthy"
// @Success  503  {object} HealthResponse "one or more services degraded"
// @Router   /health [get]
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) error {
	status, code := "ok", http.StatusOK

	dbErr := h.db.PingContext(r.Context())
	cacheErr := h.cache.Ping(r.Context())

	if dbErr != nil || cacheErr != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	return httputil.Send(w, code, HealthResponse{
		Status:  status,
		Version: h.version,
		Env:     h.env,
		DB:      dbErr == nil,
		Cache:   cacheErr == nil,
	})
}

// Live handles GET /livez
//
// @Summary  Check API liveness (process is running)
// @Tags     system
// @Produce  json
// @Success  200  {object} LiveResponse
// @Router   /livez [get]
func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) error {
	uptime := time.Since(h.startedAt).Round(time.Second).String()

	return httputil.Send(w, http.StatusOK, LiveResponse{
		Status:  "ok",
		Version: h.version,
		Env:     h.env,
		Uptime:  uptime,
	})
}

// Ready handles GET /readyz
//
// @Summary  Check API readiness (dependencies reachable)
// @Tags     system
// @Produce  json
// @Success  200  {object} ReadyResponse "ready to receive traffic"
// @Success  503  {object} ReadyResponse "not ready to receive traffic"
// @Router   /readyz [get]
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) error {
	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
	defer cancel()

	dbErr := h.db.PingContext(ctx)
	cacheErr := h.cache.Ping(ctx)

	status, code := "ready", http.StatusOK
	if dbErr != nil || cacheErr != nil {
		status, code = "not_ready", http.StatusServiceUnavailable
	}

	return httputil.Send(w, code, ReadyResponse{
		Status:  status,
		Version: h.version,
		Env:     h.env,
		Checks: ReadinessChecks{
			DB:    dbErr == nil,
			Cache: cacheErr == nil,
		},
	})
}
