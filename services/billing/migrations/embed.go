package migrations

import "embed"

// FS contains Billing database migrations.
//
//go:embed *.sql
var FS embed.FS
