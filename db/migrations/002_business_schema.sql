-- Copyright (c) 2026 Vincent Letourneau. All rights reserved.
-- Use of this source code is governed by the LICENSE file.

-- +goose Up

-- ENUMS =======================================================================

CREATE TYPE subscription_plan AS ENUM (
    'monthly',
    'annual'
);

CREATE TYPE subscription_status AS ENUM (
    'active',
    'cancelled',
    'expired'
);

CREATE TYPE purchase_type AS ENUM (
    'course',
    'bundle'
);

CREATE TYPE purchase_status AS ENUM (
    'completed',
    'refunded'
);

-- BUSINESS ====================================================================

-- Tracks user subscriptions granting full catalog access.

CREATE TABLE subscriptions (
    id           UUID                NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID                NOT NULL,
    plan         subscription_plan   NOT NULL,
    status       subscription_status NOT NULL DEFAULT 'active',
    started_at   TIMESTAMPTZ         NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ         NOT NULL,
    cancelled_at TIMESTAMPTZ         NULL,

    CONSTRAINT fk_subscriptions_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_sub_user_id     ON subscriptions (user_id);
CREATE INDEX idx_sub_user_status ON subscriptions (user_id, status, expires_at);

-- Tracks one-time purchases of a course or bundle.

CREATE TABLE purchases (
    id           UUID             NOT NULL PRIMARY KEY DEFAULT uuidv7(),
    user_id      UUID             NOT NULL,
    type         purchase_type    NOT NULL,
    item_id      UUID             NOT NULL,
    status       purchase_status  NOT NULL DEFAULT 'completed',
    amount_cents bigint           NOT NULL,
    currency     VARCHAR(3)       NOT NULL DEFAULT 'EUR',
    refunded_at  TIMESTAMPTZ      NULL,
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT now(),

    CONSTRAINT fk_purchase_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_pur_user_id ON purchases (user_id);
CREATE INDEX idx_pur_item    ON purchases (type, item_id);

ALTER TABLE course_enrollments
    ADD COLUMN purchase_id UUID NULL,
    ADD CONSTRAINT fk_enrollments_purchase_id
        FOREIGN KEY (purchase_id) REFERENCES purchases(id) ON DELETE SET NULL;

-- +goose Down

ALTER TABLE course_enrollments DROP COLUMN IF EXISTS purchase_id;

DROP INDEX IF EXISTS idx_pur_item;
DROP INDEX IF EXISTS idx_pur_user_id;
DROP TABLE IF EXISTS purchases;

DROP INDEX IF EXISTS idx_sub_user_status;
DROP INDEX IF EXISTS idx_sub_user_id;
DROP TABLE IF EXISTS subscriptions;

DROP TYPE IF EXISTS purchase_status;
DROP TYPE IF EXISTS purchase_type;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS subscription_plan;
