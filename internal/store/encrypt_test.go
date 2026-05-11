// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"testing"
	"time"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
	"github.com/nanoninja/dojo/internal/platform/security"
)

func newTestCipher(t *testing.T) *security.Cipher {
	t.Helper()
	c, err := security.NewAESCipher("12345678901234567890123456789012") // 32 bytes
	require.NoError(t, err, "setup: NewAESCipher")
	return c
}

func timePtr(s string) *time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return &t
}

// =============================================================================
// encrypt
// =============================================================================

func TestEncrypt(t *testing.T) {
	c := newTestCipher(t)

	t.Run("nil returns nil", func(t *testing.T) {
		enc, err := encrypt(c, nil)
		require.NoError(t, err)
		assert.Nil(t, enc)
	})

	t.Run("encrypts non-nil value", func(t *testing.T) {
		enc, err := encrypt(c, new("john@example.com"))
		require.NoError(t, err)
		assert.NotNil(t, enc)
	})

	t.Run("encrypt then decrypt returns original value", func(t *testing.T) {
		enc, err := encrypt(c, new("12 rue de la Paix"))
		require.NoError(t, err)
		dec, err := decrypt(c, enc)
		require.NoError(t, err)
		assert.Equal(t, "12 rue de la Paix", *dec)
	})

	t.Run("two encryptions produce different results", func(t *testing.T) {
		first, err := encrypt(c, new("john@example.com"))
		require.NoError(t, err)
		second, err := encrypt(c, new("john@example.com"))
		require.NoError(t, err)
		assert.NotEqual(t, *first, *second, "expected different ciphertexts due to random nonce")
	})
}

// =============================================================================
// decrypt
// =============================================================================

func TestDecrypt(t *testing.T) {
	c := newTestCipher(t)

	t.Run("nil returns nil", func(t *testing.T) {
		dec, err := decrypt(c, nil)
		require.NoError(t, err)
		assert.Nil(t, dec)
	})

	t.Run("corrupted value returns error", func(t *testing.T) {
		dec, err := decrypt(c, new("corrupted value"))
		assert.Error(t, err)
		assert.Nil(t, dec)
	})

	t.Run("valid encrypted value returns original", func(t *testing.T) {
		enc, err := encrypt(c, new("21 Baker Street"))
		require.NoError(t, err)
		dec, err := decrypt(c, enc)
		require.NoError(t, err)
		assert.Equal(t, "21 Baker Street", *dec)
	})
}

// =============================================================================
// encryptTime
// =============================================================================

func TestEncryptTime(t *testing.T) {
	c := newTestCipher(t)

	t.Run("nil returns nil", func(t *testing.T) {
		dec, err := encryptTime(c, nil)
		require.NoError(t, err)
		assert.Nil(t, dec)
	})

	t.Run("non-nil time returns encrypted string", func(t *testing.T) {
		enc, err := encryptTime(c, timePtr("1990-06-15"))
		require.NoError(t, err)
		assert.NotNil(t, enc)
	})
}

// =============================================================================
// decryptTime
// =============================================================================

func TestDecryptTime(t *testing.T) {
	c := newTestCipher(t)

	t.Run("nil returns nil", func(t *testing.T) {
		result, err := decryptTime(c, nil)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("valid encrypted time returns original", func(t *testing.T) {
		original := timePtr("1990-06-15")
		enc, err := encryptTime(c, original)
		require.NoError(t, err)
		result, err := decryptTime(c, enc)
		require.NoError(t, err)
		assert.True(t, result.Equal(*original))
	})

	t.Run("corrupted value returns error", func(t *testing.T) {
		result, err := decryptTime(c, new("corrupted value"))
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid date format after decryption returns error", func(t *testing.T) {
		enc, err := encrypt(c, new("not-a-date"))
		require.NoError(t, err)
		result, err := decryptTime(c, enc)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}
