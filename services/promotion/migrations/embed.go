package migrations

import "embed"

// FS contains Promotion database migrations.
//
//go:embed *.sql
var FS embed.FS
