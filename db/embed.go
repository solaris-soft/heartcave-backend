// Package migrations embeds the migrations generated from goose
// into the binary.
package migrate

import "embed"

//go:embed migrations
var Migrations embed.FS
