// Package migrations embeds the golang-migrate SQL files so cmd/migrate ships
// as a single binary with no external file dependency.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
