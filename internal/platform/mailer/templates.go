// Copyright (c) 2026 Vincent Letourneau. All rights reserved.
// Use of this source code is governed by the LICENSE file.

package mailer

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// ParseTemplates parses all HTML email templates embedded in the binary.
func ParseTemplates() *template.Template {
	return template.Must(template.ParseFS(templateFS, "templates/*.html"))
}
