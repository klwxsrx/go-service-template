package sql

type contextKey int

const (
	dbTransactionContextKey contextKey = iota
	dbTransactionLockContextKey
)
