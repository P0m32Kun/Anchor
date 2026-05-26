// Package tools provides embedded access to tool registry YAML definitions.
// These files are the single source of truth for Anchor's scanning tool schemas.
package tools

import "embed"

// FS contains all tools/*.yaml files embedded at build time.
//
//go:embed *.yaml
var FS embed.FS
