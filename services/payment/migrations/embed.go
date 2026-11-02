package migrations

import "embed"

// FS embeds the Payment service database migrations.
//
//go:embed *.sql
var FS embed.FS
