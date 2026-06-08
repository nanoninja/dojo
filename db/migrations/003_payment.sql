-- Copyright (c) 2026 Vincent Letourneau. All rights reserved.
-- Use of this source code is governed by the LICENSE file.

-- +goose Up
ALTER TABLE purchases
    ADD COLUMN provider            VARCHAR(20)  NOT NULL DEFAULT 'manual',
    ADD COLUMN provider_session_id VARCHAR(255) NULL,
    ADD COLUMN provider_payment_id VARCHAR(255) NULL;

-- +goose Down

DROP COLUMN provider,
DROP COLUMN provider_session_id,
DROP COLUMN provider_payment_id;
