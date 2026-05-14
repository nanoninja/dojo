// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/nanoninja/dojo/internal/fault"
)

// MaxBodyBytes is the maximum number of bytes read from a request body.
// It can be overridden at application startup.
var MaxBodyBytes int64 = 1 << 20 // 1MB

var validate = validator.New()

func init() {
	// strongpassword requires at least one uppercase, one lowercase and one digit.
	if err := validate.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		var hasUpper, hasLower, hasDigit bool
		for _, c := range fl.Field().String() {
			switch {
			case unicode.IsUpper(c):
				hasUpper = true
			case unicode.IsLower(c):
				hasLower = true
			case unicode.IsDigit(c):
				hasDigit = true
			}
		}
		return hasUpper && hasLower && hasDigit
	}); err != nil {
		panic("registering strongpassword validator: " + err.Error())
	}
}

// Bind decodes the JSON request body into v.
// It limits the body size to MaxBodyBytes to prevent memory exhaustion.
// Returns a fault.BadRequest if the body is missing, malformed, or exceeds the limit.
func Bind(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return fault.BadRequest("invalid request body", err)
	}

	// Reject trailing JSON tokens (e.g. "{}{}").
	// This keeps the API strict and prevents clients from sending multiple payloads.
	if dec.Decode(&struct{}{}) != io.EOF {
		return fault.BadRequest("invalid request body", nil)
	}

	if err := validate.Struct(v); err != nil {
		return fault.BadRequest(err.Error(), err)
	}

	return nil
}

// ValidateUUID reports whether id is a valid UUID.
func ValidateUUID(id string) bool {
	return validate.Var(id, "required,uuid") == nil
}
