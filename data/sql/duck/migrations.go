package duck

import "embed"

//go:embed *.sql
var SQLMigrations embed.FS
