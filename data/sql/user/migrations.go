package user

import (
	"embed"

	"github.com/klwxsrx/go-service-template/pkg/sql"
)

var Migrations = sql.FSMigrations(migrationFiles)

//go:embed *.sql
var migrationFiles embed.FS
