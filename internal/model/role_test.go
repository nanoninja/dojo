// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package model_test

import (
	"database/sql"
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/model"
)

func TestParseRole(t *testing.T) {
	tests := []struct {
		input string
		want  model.Role
	}{
		{"user", model.RoleUser},
		{"instructor", model.RoleInstructor},
		{"moderator", model.RoleModerator},
		{"manager", model.RoleManager},
		{"admin", model.RoleAdmin},
		{"superadmin", model.RoleSuperAdmin},
		{"system", model.RoleSystem},
		{"unknown", model.RoleUser},
		{"", model.RoleUser},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := model.ParseRole(tt.input)
			assert.Equalf(t, tt.want, got, "ParseRole(%q)", tt.input)
		})
	}
}

func TestRoleString(t *testing.T) {
	tests := []struct {
		role model.Role
		want string
	}{
		{model.RoleUser, "user"},
		{model.RoleInstructor, "instructor"},
		{model.RoleModerator, "moderator"},
		{model.RoleManager, "manager"},
		{model.RoleAdmin, "admin"},
		{model.RoleSuperAdmin, "superadmin"},
		{model.RoleSystem, "system"},
		{model.Role(999), "user"}, // unknown defaults to user
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.role.String()
			assert.Equalf(t, tt.want, got, "Role(%d).String()", tt.role)
		})
	}
}

func TestRoleHierarchy(t *testing.T) {
	// Verify the numeric ordering is correct.
	assert.True(t, model.RoleUser < model.RoleModerator, "RoleUser should be less than RoleModerator")
	assert.True(t, model.RoleInstructor < model.RoleModerator, "RoleInstructor should be less than RoleModerator")
	assert.True(t, model.RoleModerator < model.RoleManager, "RoleModerator should be less than RoleManager")
	assert.True(t, model.RoleManager < model.RoleAdmin, "RoleManager should be less than RoleAdmin")
	assert.True(t, model.RoleAdmin < model.RoleSuperAdmin, "RoleAdmin should be less than RoleSuperAdmin")
	assert.True(t, model.RoleSuperAdmin < model.RoleSystem, "RoleSuperAdmin should be less than RoleSystem")
}

func TestRoleScan(t *testing.T) {
	t.Run("scan from []byte", func(t *testing.T) {
		var r model.Role

		require.NoError(t, r.Scan([]byte("admin")))
		assert.Equal(t, model.RoleAdmin, r, "Scan([]byte)")
	})

	t.Run("scan from string", func(t *testing.T) {
		var r model.Role

		require.NoError(t, r.Scan("moderator"))
		assert.Equal(t, model.RoleModerator, r, "Scan(string)")
	})

	t.Run("scan nil defaults to user", func(t *testing.T) {
		var r model.Role

		require.NoError(t, r.Scan(nil))
		assert.Equal(t, model.RoleUser, r, "Scan(nil)")
	})

	t.Run("scan unsupported type returns error", func(t *testing.T) {
		var r model.Role

		assert.Error(t, r.Scan(42))
	})

	t.Run("implements sql.Scanner", func(_ *testing.T) {
		// Compile-time check that *Role satisfies sql.Scanner.
		var _ sql.Scanner = (*model.Role)(nil)
	})
}
