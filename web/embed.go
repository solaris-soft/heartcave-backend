// Package web holds the embedded template filesystem.
package web

import "embed"

//go:embed templates
var Templates embed.FS
