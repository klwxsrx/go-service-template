package duck

import (
	"embed"

	pkgsql "github.com/klwxsrx/go-service-template/pkg/sql"
)

var Migrations = pkgsql.FSMigrations(migrationFiles)

//go:embed *.sql
var migrationFiles embed.FS
