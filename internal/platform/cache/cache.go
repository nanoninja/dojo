// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

// Package cache provides a Redis client wrapper.
package cache

import (
	"context"
	"fmt"

	"github.com/nanoninja/dojo/internal/config"
	"github.com/redis/go-redis/v9"
)

// Client wraps redis.Client to provide a configured Redis connection.
type Client struct {
	*redis.Client
}

// Ping checks that the Redis connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx).Err()
}

// Open creates a new Redis client and verifies the connection with a ping.
func Open(cfg config.Redis) (*Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	if err := c.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}
	return &Client{c}, nil
}
