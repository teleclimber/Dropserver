package domain

import "errors"

// ErrNoRowsAffected indicates a no DB rows were affected
var ErrNoRowsAffected = errors.New("No rows affected")

// ErrNoRowsInResultSet indicates that no rows were found in db
var ErrNoRowsInResultSet = errors.New("No rows in result set")

// ErrAppspaceLockedClosed the operation could not beperformed because the
// appspace is currently closed
var ErrAppspaceLockedClosed = errors.New("appspace is locked closed")

// ErrAppVersionInUse indicates the operation could not be performed
// on the app version because something (probably an appspace)
// depends on it
var ErrAppVersionInUse = errors.New("app version in use")
