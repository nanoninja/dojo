// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package security

import "errors"

var (
	// ErrInvalidKeySize is returned when the encryption key is not exactly 32 bytes.
	ErrInvalidKeySize = errors.New("encryption key must be exactly 32 bytes")

	// ErrCiphertextTooShort is returned when the ciphertext is shorter than the nonce.
	ErrCiphertextTooShort = errors.New("ciphertext too short")

	// ErrDecryptionFailed is returned when GCM authentication or decryption fails.
	ErrDecryptionFailed = errors.New("decryption failed")
)
