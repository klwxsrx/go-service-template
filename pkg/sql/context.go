package sql

type contextKey int

const (
	dbConnectionContextKey contextKey = iota
	dbTransactionContextKey
	dbTransactionLockContextKey
)
