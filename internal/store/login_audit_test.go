// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/dojo/internal/model"
	"github.com/nanoninja/dojo/internal/store"
	"github.com/nanoninja/dojo/internal/testutil"
)

// newTestAuditLog returns a minimal login audit log for a given user.
func newTestAuditLog(userID *string, status model.LoginStatus) *model.LoginAuditLog {
	return &model.LoginAuditLog{
		UserID:    userID,
		Email:     "john.doe@example.com",
		IPAddress: "1.2.3.4",
		UserAgent: "Go-test-client/1.0",
		Status:    status,
	}
}

// setupAuditStore creates a user and returns the audit store and user ID for use in tests.
func setupAuditStore(t *testing.T) (store.LoginAuditStore, string) {
	t.Helper()
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "login_audit_logs", "users")

	cipher := testutil.NewTestCipher(t)
	us := store.NewUserStore(db, cipher)
	as := store.NewLoginAuditStore(db, cipher)
	u := newTestUser()

	assert.NoError(t, us.Create(context.Background(), u), "setup: Create() user")

	return as, u.ID
}

// ============================================================================
// Create
// ============================================================================

func TestLoginAuditStore_Create_WithUser(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()
	log := newTestAuditLog(&userID, model.LoginStatusSuccess)

	assert.NoError(t, as.Create(ctx, log))
	assert.NotEqual(t, "", log.ID, "Create() dit not set ID")
}

func TestLoginAuditStore_Create_WithoutUser(t *testing.T) {
	db := testutil.OpenTestDB(t)
	testutil.TruncateTable(t, db, "login_audit_logs", "users")
	cipher := testutil.NewTestCipher(t)
	as := store.NewLoginAuditStore(db, cipher)
	ctx := context.Background()

	// user_id is nil: attempt on an unknown email.
	log := newTestAuditLog(nil, model.LoginStatusFailedNotFound)

	assert.NoError(t, as.Create(ctx, log))
	assert.NotEqual(t, "", log.ID, "Create() did not set ID")
}

// ============================================================================
// FindByUser
// ============================================================================

func TestLoginAuditStore_FindByUser_ReturnsLogs(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	statuses := []model.LoginStatus{
		model.LoginStatusSuccess,
		model.LoginStatusFailedPassword,
		model.LoginStatusFailedLocked,
	}
	for _, s := range statuses {
		assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, s)), "setup: Create()")
	}

	logs, err := as.FindByUser(ctx, userID, 10)

	assert.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestLoginAuditStore_FindByUser_DecryptsFields(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)))
	logs, err := as.FindByUser(ctx, userID, 10)

	assert.NoError(t, err)
	assert.NotEmpty(t, logs, "FindByUser() returned no logs")
	assert.Equal(t, "john.doe@example.com", logs[0].Email)
	assert.Equal(t, "1.2.3.4", logs[0].IPAddress)
}

func TestLoginAuditStore_FindByUser_RespectsLimit(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	for range 5 {
		assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)), "setup: Create()")
	}

	logs, err := as.FindByUser(ctx, userID, 3)
	assert.NoError(t, err)
	assert.Len(t, logs, 3, "limit enforced")
}

func TestLoginAuditStore_FindByUser_Empty(t *testing.T) {
	as, userID := setupAuditStore(t)

	logs, err := as.FindByUser(context.Background(), userID, 10)

	assert.NoError(t, err)
	assert.Empty(t, logs)
}

// ============================================================================
// List
// ============================================================================

func TestLoginAuditStore_List_ReturnsAll(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	for _, s := range []model.LoginStatus{
		model.LoginStatusSuccess,
		model.LoginStatusFailedPassword,
		model.LoginStatusFailedLocked,
	} {
		assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, s)), "setup: Create()")
	}

	logs, total, err := as.List(ctx, store.AuditFilter{Limit: 10})

	assert.NoError(t, err)
	assert.Equal(t, 3, int64(total))
	assert.Len(t, logs, 3)
}

func TestLoginAuditStore_List_FilterByUserID(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	// One log for the known user, one anonymous.
	_ = as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess))
	_ = as.Create(ctx, newTestAuditLog(nil, model.LoginStatusFailedNotFound))

	logs, total, err := as.List(ctx, store.AuditFilter{UserID: &userID, Limit: 10})

	assert.NoError(t, err)
	assert.Equal(t, 1, int64(total))
	assert.Len(t, logs, 1)
}

func TestLoginAuditStore_List_FilterByStatus(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	_ = as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess))
	_ = as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusFailedPassword))
	_ = as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusFailedPassword))

	logs, total, err := as.List(ctx, store.AuditFilter{
		Status: model.LoginStatusFailedPassword,
		Limit:  10,
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, int64(total))
	assert.Len(t, logs, 2)
}

func TestLoginAuditStore_List_FilterBySince(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	_ = as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess))

	// since in the future → nothing qualifies
	logs, total, err := as.List(ctx, store.AuditFilter{
		Since: timeNowPlusDuration(time.Minute),
		Limit: 10,
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, int64(total), "since is in the future")
	assert.Empty(t, logs)
}

func TestLoginAuditStore_List_FilterByUntil(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	_ = as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess))

	// until in the past → nothing qualifies
	logs, total, err := as.List(ctx, store.AuditFilter{
		Until: timeNowPlusDuration(-time.Minute),
		Limit: 10,
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, int64(total), "until is in the past")
	assert.Empty(t, logs)
}

func TestLoginAuditStore_List_Pagination(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	for range 5 {
		assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)), "setup: Create()")
	}

	logs, total, err := as.List(ctx, store.AuditFilter{Limit: 2, Offset: 2})

	assert.NoError(t, err)
	assert.Equal(t, 5, int64(total), "full count unaffected by pagination")
	assert.Len(t, logs, 2, "limit enforced")
}

func TestLoginAuditStore_List_DecryptsFields(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)), "setup: Create()")

	logs, _, err := as.List(ctx, store.AuditFilter{Limit: 10})

	assert.NoError(t, err)
	assert.NotEmpty(t, logs, "List() returned no logs")
	assert.Equal(t, "john.doe@example.com", logs[0].Email)
	assert.Equal(t, "1.2.3.4", logs[0].IPAddress)
}

// ============================================================================
// Purge
// ============================================================================

func TestLoginAuditStore_Purge_DeletesOldEntries(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	for range 3 {
		assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)), "setup: Create()")
	}

	// retention=0 → created_at < NOW() matches all existing rows
	deleted, err := as.Purge(ctx, 0, 100)

	assert.NoError(t, err)
	assert.Equal(t, 3, int64(deleted))
}

func TestLoginAuditStore_Purge_RespectsRetention(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)), "setup: Create()")

	// retention=90 days → row just inserted is not old enough
	deleted, err := as.Purge(ctx, 90*24*time.Hour, 100)

	assert.NoError(t, err)
	assert.Equal(t, 0, int64(deleted), "row too recent")
}

func TestLoginAuditStore_Purge_RespectsLimit(t *testing.T) {
	as, userID := setupAuditStore(t)
	ctx := context.Background()

	for range 5 {
		assert.NoError(t, as.Create(ctx, newTestAuditLog(&userID, model.LoginStatusSuccess)), "setup: Create()")
	}

	// batchSize=2 → at most 2 rows deleted per call
	deleted, err := as.Purge(ctx, 0, 2)

	assert.NoError(t, err)
	assert.Equal(t, 2, int64(deleted), "batch limit enforced")
}

// timeNowPlusDuration returns time.Now() offset by d, used to build test time boundaries.
func timeNowPlusDuration(d time.Duration) time.Time {
	return time.Now().Add(d)
}
