package domain

import "errors"

// ErrNoRowsAffected indicates a no DB rows were affected
var ErrNoRowsAffected = errors.New("No rows affected")

// ErrNoRowsInResultSet indicates that no rows were found in db
var ErrNoRowsInResultSet = errors.New("No rows in result set")
