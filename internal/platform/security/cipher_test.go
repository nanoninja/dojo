// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package security

import (
	"testing"

	"github.com/nanoninja/assert"
	"github.com/nanoninja/assert/require"
)

const validKey = "12345678901234567890123456789012" // 32 bytes

func TestNewAESCipher(t *testing.T) {
	t.Run("valid 32-byte key", func(t *testing.T) {
		c, err := NewAESCipher(validKey)
		require.NoError(t, err)
		require.NotNil(t, c)
	})

	t.Run("key too short", func(t *testing.T) {
		_, err := NewAESCipher("tooshort")
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("key too long", func(t *testing.T) {
		_, err := NewAESCipher(validKey + "x")
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := NewAESCipher("")
		assert.ErrorIs(t, err, ErrInvalidKeySize)
	})
}

func TestEncryptDecrypt(t *testing.T) {
	c, err := NewAESCipher(validKey)
	require.NoError(t, err, "setup: NewAESCipher")

	t.Run("encrypt then decrypt returns original text", func(t *testing.T) {
		plaintext := "sensitive data"

		encrypted, err := c.Encrypt(plaintext)
		require.NoError(t, err)

		decrypted, err := c.Decrypt(encrypted)
		require.NoError(t, err)

		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("two encryptions of the same text produce different results", func(t *testing.T) {
		plaintext := "sensitive data"

		first, err := c.Encrypt(plaintext)
		require.NoError(t, err)

		second, err := c.Encrypt(plaintext)
		require.NoError(t, err)

		assert.NotEqual(t, first, second, "expected different ciphertexts due to random nonce")
	})

	t.Run("empty plaintext", func(t *testing.T) {
		encrypted, err := c.Encrypt("")
		require.NoError(t, err)

		decrypted, err := c.Decrypt(encrypted)
		require.NoError(t, err)

		assert.Equal(t, "", decrypted)
	})
}

func TestRandomToken(t *testing.T) {
	t.Run("returns hex string of correct length", func(t *testing.T) {
		token, err := RandomToken(32)
		require.NoError(t, err)
		// 32 bytes → 64 hex characters
		assert.Equal(t, 64, len(token))
	})

	t.Run("two tokens are different", func(t *testing.T) {
		first, err := RandomToken(32)
		require.NoError(t, err)
		second, err := RandomToken(32)
		require.NoError(t, err)
		assert.NotEqual(t, first, second, "expected different tokens, got identical values")
	})

	t.Run("returns only hex characters", func(t *testing.T) {
		token, err := RandomToken(16)
		require.NoError(t, err)
		for _, c := range token {
			isDigit := '0' <= c && c <= '9'
			isHexLetter := 'a' <= c && c <= 'f'
			assert.Truef(t, isDigit || isHexLetter, "unexpected character %q in token %q", c, token)
		}
	})
}

func TestDecryptErrors(t *testing.T) {
	c, err := NewAESCipher(validKey)
	require.NoError(t, err, "setup: NewAESCipher")

	t.Run("invalid base64", func(t *testing.T) {
		_, err := c.Decrypt("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("ciphertext too short", func(t *testing.T) {
		_, err := c.Decrypt("dG9vc2hvcnQ=") // base64("tooshort")
		assert.ErrorIs(t, err, ErrCiphertextTooShort)
	})

	t.Run("altered ciphertext", func(t *testing.T) {
		encrypted, err := c.Encrypt("sensitive data")
		require.NoError(t, err)

		// Alter the last base64 character.
		altered := encrypted[:len(encrypted)-1] + "X"

		_, err = c.Decrypt(altered)
		assert.ErrorIs(t, err, ErrDecryptionFailed)
	})
}
