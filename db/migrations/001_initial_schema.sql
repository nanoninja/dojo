-- +goose Up


-- ENUMS =======================================================

CREATE TYPE user_status AS ENUM (
    'pending',
    'active',
    'suspended',
    'banned',
    'deleted'
);

CREATE TYPE user_role AS ENUM (
    'user',
    'instructor',
    'moderator',
    'manager',
    'admin',
    'superadmin',
    'system'
);

CREATE TYPE verification_token_type AS ENUM (
    'email_verification',
    'password_reset',
    'otp'
);

CREATE TYPE login_status AS ENUM (
    'success',
    'failed_password',
    'failed_locked',
    'failed_not_found',
    'failed_unverified'
);

CREATE TYPE course_level AS ENUM (
    'beginner',
    'intermediate',
    'advanced',
    'expert'
);

CREATE TYPE content_type AS ENUM (
    'video',
    'article',
    'audio',
    'live',
    'document',
    'mixed'
);

CREATE TYPE enrollment_status AS ENUM (
    'active',
    'completed',
    'expired'
    'refunded'
);

-- FUNCTIONS ===================================================

-- +goose StatementBegin

-- Sets updated_at to NOW() on every UPDATE.
-- Used by all tables that have an updated_at column via a BEFORE UPDATE trigger.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Recomputes duration_minutes on chapters after any lesson duration change.
-- Triggered by sync_chapter_duration_on_lesson (AFTER INSERT, UPDATE OF duration_minutes, DELETE on lessons).
CREATE OR REPLACE FUNCTION sync_chapter_duration()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE chapters
    SET duration_minutes = (
        SELECT COALESCE(SUM(duration_minutes), 0)
        FROM lessons
        WHERE chapter_id = COALESCE(NEW.chapter_id, OLD.chapter_id)
    )
    WHERE id = COALESCE(NEW.chapter_id, OLD.chapter_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Recomputes duration_minutes on courses after a chapter duration changes.
-- Triggered by sync_course_duration_on_chapter (AFTER UPDATE OF duration_minutes on chapters).
CREATE OR REPLACE FUNCTION sync_course_duration()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE courses
    SET duration_minutes = (
        SELECT COALESCE(SUM(duration_minutes), 0)
        FROM chapters
        WHERE course_id = NEW.course_id
    )
    WHERE id = NEW.course_id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Recomputes course_count on categories after an assignment is added or removed.
-- Triggered by sync_category_course_count_on_assignment (AFTER INSERT, DELETE on courses_categories).
CREATE OR REPLACE FUNCTION sync_category_course_count()
RETURNS TRIGGER AS $$
DECLARE
    target_id UUID;
BEGIN
    target_id := COALESCE(NEW.category_id, OLD.category_id);
    UPDATE categories
    SET course_count = (
        SELECT COUNT(*)
        FROM courses_categories
        WHERE category_id = target_id
    )
    WHERE id = target_id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- +goose StatementEnd


-- USERS =======================================================

CREATE TABLE users (
    -- Identity
    id                    UUID           NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    email                 VARCHAR(160)   NOT NULL UNIQUE,
    password_hash         VARCHAR(255)   NOT NULL,
    status                user_status    NOT NULL DEFAULT 'pending',
    is_verified           BOOLEAN        NOT NULL DEFAULT false,
    is_2fa_enabled        BOOLEAN        NOT NULL DEFAULT false,
    role                  user_role      NOT NULL DEFAULT 'user',

    -- Profile
    first_name            VARCHAR(100)   DEFAULT NULL,
    last_name             VARCHAR(100)   DEFAULT NULL,
    company_name          VARCHAR(100)   DEFAULT NULL,
    headline              VARCHAR(150)   DEFAULT NULL,
    bio                   TEXT,
    avatar_url            VARCHAR(512)   DEFAULT NULL,
    website               VARCHAR(255)   DEFAULT NULL,

    -- Address
    address_line1         VARCHAR(512)   DEFAULT NULL,
    address_line2         VARCHAR(512)   DEFAULT NULL,
    city                  VARCHAR(100)   DEFAULT NULL,
    postal_code           VARCHAR(20)    DEFAULT NULL,
    latitude              DECIMAL(10,8)  DEFAULT NULL,
    longitude             DECIMAL(11,8)  DEFAULT NULL,

    -- Business
    vat_number            VARCHAR(128)   DEFAULT NULL,
    country_code          CHAR(2)        DEFAULT NULL,

    -- Locale
    language              VARCHAR(10)    NOT NULL DEFAULT 'fr-FR',
    timezone              VARCHAR(50)    NOT NULL DEFAULT 'Europe/Paris',
    birth_date            VARCHAR(64)    DEFAULT NULL,

    -- Activity
    last_login_at         TIMESTAMPTZ    DEFAULT NULL,
    last_login_ip         VARCHAR(128)   DEFAULT NULL,
    login_count           INTEGER        NOT NULL DEFAULT 0,
    failed_login_attempts INT            NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ,

    -- Timestamps
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ    DEFAULT NULL,
    banned_at             TIMESTAMPTZ    DEFAULT NULL,
    deleted_at            TIMESTAMPTZ    DEFAULT NULL
);

CREATE INDEX idx_users_status ON users (status) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_created_at ON users (created_at) WHERE deleted_at IS NULL;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();


-- VERIFICATION TOKENS =========================================

CREATE TABLE verification_tokens (
    id         UUID                    NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    user_id    UUID                    NOT NULL,
    token      VARCHAR(255)            NOT NULL,
    type       verification_token_type NOT NULL,
    attempts   SMALLINT                NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ             NOT NULL,
    used_at    TIMESTAMPTZ             DEFAULT NULL,
    created_at TIMESTAMPTZ             NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_verification_tokens_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes

CREATE INDEX idx_vts_lookup ON verification_tokens (user_id, token, type)
WHERE used_at IS NULL;

CREATE INDEX idx_vts_expiry ON verification_tokens (expires_at);


-- REFRESH TOKENS ==============================================

CREATE TABLE refresh_tokens (
    id         UUID         NOT NULL DEFAULT uuidv7() PRIMARY KEY,
    user_id    UUID         NOT NULL,
    token_hash VARCHAR(64)  NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ  DEFAULT NULL,

    CONSTRAINT fk_refresh_tokens_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);


-- LOGIN AUDIT LOGS ============================================

CREATE TABLE login_audit_logs (
    id         UUID         NOT NULL DEFAULT uuidv7(),
    user_id    UUID         NULL,
    email      VARCHAR(256) NOT NULL DEFAULT '',
    ip_address VARCHAR(128) NOT NULL DEFAULT '',
    user_agent TEXT         NOT NULL DEFAULT '',
    status     login_status NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_login_audit_logs PRIMARY KEY (id),
    CONSTRAINT fk_login_audit_logs_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX idx_lal_user_id    ON login_audit_logs (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX idx_lal_created_at ON login_audit_logs (created_at);


-- CATALOG & CONTENT ===========================================

CREATE TABLE courses (
    -- Identity
    id                  UUID           PRIMARY KEY NOT NULL DEFAULT uuidv7(),
    instructor_id       UUID           NOT NULL,
    slug                VARCHAR(255)   UNIQUE NOT NULL,

    -- Content
    title               VARCHAR(255)   NOT NULL,
    subtitle            VARCHAR(255)   DEFAULT NULL,
    description         TEXT,
    prerequisites       TEXT,
    objectives          TEXT,

    -- SEO
    meta_title          VARCHAR(255)   DEFAULT NULL,
    meta_description    VARCHAR(255)   DEFAULT NULL,
    meta_keywords       VARCHAR(255)   DEFAULT NULL,

    -- Media
    thumbnail_url       VARCHAR(512)   DEFAULT NULL,
    trailer_url         VARCHAR(512)   DEFAULT NULL,

    -- Classification
    level               course_level   DEFAULT 'beginner',
    content_type        content_type   DEFAULT 'video',
    language            VARCHAR(10)    DEFAULT 'fr-FR',
    duration_minutes    INTEGER        DEFAULT 0,

    -- Pricing
    is_free             BOOLEAN        DEFAULT FALSE,
    subscription_only   BOOLEAN        DEFAULT FALSE,
    price_cents         INTEGER        DEFAULT 0,
    currency            CHAR(3)        DEFAULT 'EUR',

    -- Visibility
    is_published        BOOLEAN        DEFAULT FALSE,
    is_featured         BOOLEAN        DEFAULT FALSE,
    certificate_enabled BOOLEAN        DEFAULT FALSE,
    sort_order          SMALLINT       DEFAULT 0,

    -- Stats (denormalized)
    student_count       INTEGER        DEFAULT 0,
    rating_average      DECIMAL(3, 2)  DEFAULT 0.00,
    rating_count        INTEGER        DEFAULT 0,

    -- Timestamps
    published_at        TIMESTAMPTZ    DEFAULT NULL,
    created_at          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ    DEFAULT NULL,
    deleted_at          TIMESTAMPTZ    DEFAULT NULL,

    CONSTRAINT fk_courses_instructor_id
        FOREIGN KEY (instructor_id) REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX idx_courses_instructor_id ON courses (instructor_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_courses_is_published  ON courses (is_published)  WHERE deleted_at IS NULL;
CREATE INDEX idx_courses_level         ON courses (level)         WHERE deleted_at IS NULL;

CREATE TRIGGER update_courses_updated_at
    BEFORE UPDATE ON courses
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE chapters (
    id               UUID         PRIMARY KEY NOT NULL DEFAULT uuidv7(),
    course_id        UUID         NOT NULL,
    title            VARCHAR(255) NOT NULL,
    slug             VARCHAR(255) NOT NULL,
    description      TEXT         DEFAULT NULL,
    sort_order       SMALLINT     NOT NULL DEFAULT 10,
    is_free          BOOLEAN      DEFAULT FALSE,
    is_published     BOOLEAN      DEFAULT FALSE,
    duration_minutes INTEGER      DEFAULT 0,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  DEFAULT NULL,

    CONSTRAINT uq_chapters_course_slug
        UNIQUE (course_id, slug),
    CONSTRAINT fk_chapters_course_id
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

CREATE TRIGGER update_chapters_updated_at
    BEFORE UPDATE ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER sync_course_duration_on_chapter
    AFTER UPDATE OF duration_minutes ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION sync_course_duration();

CREATE TABLE lessons (
    id               UUID PRIMARY KEY NOT NULL DEFAULT uuidv7(),
    chapter_id       UUID             NOT NULL,
    title            VARCHAR(255)     NOT NULL,
    slug             VARCHAR(255)     NOT NULL,
    description      TEXT,
    sort_order       SMALLINT         NOT NULL DEFAULT 0,
    content_type     content_type     NOT NULL DEFAULT 'video',
    media_url        VARCHAR(512)     DEFAULT NULL,
    is_free          BOOLEAN          DEFAULT FALSE,
    is_published     BOOLEAN          DEFAULT FALSE,
    duration_minutes INTEGER          DEFAULT 0,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ      DEFAULT NULL,

    CONSTRAINT uq_lessons_chapter_id
        UNIQUE (chapter_id, slug),
    CONSTRAINT fk_lessons_chapter_id
        FOREIGN KEY (chapter_id) REFERENCES chapters(id) ON DELETE CASCADE
);

CREATE INDEX idx_lessons_chapter_id ON lessons (chapter_id);

CREATE TRIGGER update_lessons_updated_at
    BEFORE UPDATE ON lessons
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER sync_chapter_duration_on_lesson
    AFTER INSERT OR UPDATE OF duration_minutes OR DELETE ON lessons
    FOR EACH ROW
    EXECUTE FUNCTION sync_chapter_duration();

CREATE TABLE lesson_resources (
    id              UUID         PRIMARY KEY NOT NULL DEFAULT uuidv7(),
    lesson_id       UUID         NOT NULL,
    title           VARCHAR(255) NOT NULL,
    description     VARCHAR(255) DEFAULT NULL,
    file_url        VARCHAR(512) NOT NULL,
    file_name       VARCHAR(255) NOT NULL,
    file_size_bytes BIGINT       DEFAULT NULL,
    mime_type       VARCHAR(100) DEFAULT NULL,
    sort_order      SMALLINT     DEFAULT 10,
    is_public       BOOLEAN      DEFAULT FALSE,
    download_count  INTEGER      DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_lesson_resources_lesson_id
        FOREIGN KEY (lesson_id) REFERENCES lessons(id) ON DELETE CASCADE
);

CREATE TABLE categories (
    -- Identity
    id           UUID      PRIMARY KEY NOT NULL DEFAULT uuidv7(),
    parent_id    UUID      DEFAULT NULL,
    slug         VARCHAR(120) UNIQUE NOT NULL,

    -- Content
    name         VARCHAR(100) NOT NULL,
    description  TEXT         DEFAULT NULL,

    -- Display
    color_hex    CHAR(9)      DEFAULT NULL,
    icon_url     VARCHAR(255) DEFAULT NULL,
    sort_order   SMALLINT     NOT NULL DEFAULT 10,
    is_visible   BOOLEAN      DEFAULT TRUE,

    -- Stats (denormalized)
    course_count INTEGER      DEFAULT 0,

    -- Timestamps
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  DEFAULT NULL,
    deleted_at   TIMESTAMPTZ  DEFAULT NULL,

    CONSTRAINT fk_categories_parent_id
        FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE TRIGGER update_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE courses_categories (
    course_id   UUID        NOT NULL,
    category_id UUID        NOT NULL,
    is_primary  BOOLEAN NOT NULL DEFAULT FALSE,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (course_id, category_id),
    CONSTRAINT fk_courses_categories_course_id
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    CONSTRAINT fk_courses_categories_category_id
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE TRIGGER sync_category_course_count_on_assignment
    AFTER INSERT OR DELETE ON courses_categories
    FOR EACH ROW
    EXECUTE FUNCTION sync_category_course_count();

CREATE TABLE tags (
    id         UUID PRIMARY KEY   NOT NULL DEFAULT uuidv7(),
    slug       VARCHAR(80) UNIQUE NOT NULL,
    name       VARCHAR(80)        NOT NULL,
    created_at TIMESTAMPTZ        NOT NULL DEFAULT NOW()
);

CREATE TABLE courses_tags (
    course_id UUID NOT NULL,
    tag_id    UUID NOT NULL,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (course_id, tag_id),

    CONSTRAINT fk_courses_tags_course_id
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    CONSTRAINT fk_courses_tags_tag_id
        FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- ENROLLMENTS =================================================

CREATE TABLE course_enrollments (
    id               UUID              PRIMARY KEY NOT NULL DEFAULT uuidv7(),
    user_id          UUID              NOT NULL,
    course_id        UUID              NOT NULL,
    status           enrollment_status NOT NULL DEFAULT 'active',
    progress_percent NUMERIC(5,2)      NOT NULL DEFAULT 0.00,
    last_accessed_at TIMESTAMPTZ       NULL,
    enrolled_at      TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    completed_at     TIMESTAMPTZ       NULL,
    expires_at       TIMESTAMPTZ       NULL,

    CONSTRAINT uq_course_enrollments_user_course UNIQUE (user_id, course_id),

    CONSTRAINT fk_course_enrollments_user_id
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_course_enrollments_course_id
        FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

CREATE INDEX idx_ce_user_id   ON course_enrollments (user_id);
CREATE INDEX idx_ce_course_id ON course_enrollments (course_id);
CREATE INDEX idx_ce_status    ON course_enrollments (status);

-- GOOSE DOWN ==================================================

-- +goose Down
DROP INDEX IF EXISTS idx_lal_created_at;
DROP INDEX IF EXISTS idx_lal_user_id;
DROP INDEX IF EXISTS idx_vts_expiry;
DROP INDEX IF EXISTS idx_vts_lookup;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_courses_instructor_id;
DROP INDEX IF EXISTS idx_courses_is_published;
DROP INDEX IF EXISTS idx_courses_level;
DROP INDEX IF EXISTS idx_lessons_chapter_id;
DROP INDEX IF EXISTS idx_ce_status;
DROP INDEX IF EXISTS idx_ce_course_id;
DROP INDEX IF EXISTS idx_ce_user_id;

DROP TRIGGER IF EXISTS sync_category_course_count_on_assignment ON courses_categories;
DROP TRIGGER IF EXISTS sync_chapter_duration_on_lesson ON lessons;
DROP TRIGGER IF EXISTS sync_course_duration_on_chapter ON chapters;
DROP TRIGGER IF EXISTS update_categories_updated_at ON categories;
DROP TRIGGER IF EXISTS update_lessons_updated_at ON lessons;
DROP TRIGGER IF EXISTS update_chapters_updated_at ON chapters;
DROP TRIGGER IF EXISTS update_courses_updated_at ON courses;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP TABLE IF EXISTS course_enrollments;
DROP TABLE IF EXISTS courses_tags;
DROP TABLE IF EXISTS courses_categories;
DROP TABLE IF EXISTS login_audit_logs;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS verification_tokens;
DROP TABLE IF EXISTS lesson_resources;
DROP TABLE IF EXISTS lessons;
DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS sync_category_course_count;
DROP FUNCTION IF EXISTS sync_course_duration;
DROP FUNCTION IF EXISTS sync_chapter_duration;
DROP FUNCTION IF EXISTS update_updated_at_column;

DROP TYPE IF EXISTS verification_token_type;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS login_status;
DROP TYPE IF EXISTS course_level;
DROP TYPE IF EXISTS content_type;
DROP TYPE IF EXISTS enrollment_status;
