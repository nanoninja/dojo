// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package store

import (
	"fmt"
	"time"

	"github.com/nanoninja/dojo/internal/platform/security"
)

func encrypt(c *security.Cipher, val *string) (*string, error) {
	if val == nil {
		return nil, nil
	}
	enc, err := c.Encrypt(*val)
	if err != nil {
		return nil, fmt.Errorf("encrypting field: %w", err)
	}
	return &enc, nil
}

func decrypt(c *security.Cipher, val *string) (*string, error) {
	if val == nil {
		return nil, nil
	}
	dec, err := c.Decrypt(*val)
	if err != nil {
		return nil, fmt.Errorf("decrypting field: %w", err)
	}
	return &dec, nil
}

func encryptTime(c *security.Cipher, t *time.Time) (*string, error) {
	if t == nil {
		return nil, nil
	}
	formatted := t.Format(time.DateOnly)
	return encrypt(c, &formatted)
}

func decryptTime(c *security.Cipher, val *string) (*time.Time, error) {
	dec, err := decrypt(c, val)
	if err != nil {
		return nil, err
	}
	if dec == nil {
		return nil, nil
	}
	t, err := time.Parse(time.DateOnly, *dec)
	if err != nil {
		return nil, fmt.Errorf("parsing decrypted time: %w", err)
	}
	return &t, nil
}
