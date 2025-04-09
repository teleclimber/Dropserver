package domain

import "errors"

// ErrNoRowsAffected indicates a no DB rows were affected
var ErrNoRowsAffected = errors.New("no rows affected")

// ErrNoRowsInResultSet indicates that no rows were found in db
var ErrNoRowsInResultSet = errors.New("no rows in result set")

// ErrUniqueConstraintViolation indicates that a unique constraint
// such as an index or primary key was violated
var ErrUniqueConstraintViolation = errors.New("unique constraint violation")

// ErrStorageExceeded indicates that a storage quota has been or will be exceeded
var ErrStorageExceeded = errors.New("storage limit reached")

// ErrIdentifierExists is returned when a user email or other unique identifier already exists
var ErrIdentifierExists = errors.New("identifier exists")

// ErrBadAuth is returned when a user name / passwordcombintaion is incorrect
var ErrBadAuth = errors.New("authentication incorrect")

// ErrAppspaceLockedClosed the operation could not beperformed because the
// appspace is currently closed
var ErrAppspaceLockedClosed = errors.New("appspace is locked closed")

// ErrLocationKeyDoesNotExist means that an attempt to open a path at a location key
// failed because the location key itself was not found on disk
var ErrLocationKeyDoesNotExist = errors.New("location key was not found on disk")

// ErrAppManifestNotFound means the application manifest (dropapp.json) file was not found
var ErrAppManifestNotFound = errors.New("app manifest dropapp.json not found")

// ErrAppVersionInUse indicates the operation could not be performed
// on the app version because something (probably an appspace)
// depends on it
var ErrAppVersionInUse = errors.New("app version in use")

// ErrTokenNotFound is used when a token is ued to gain access to a
// process or to data. This error indicates that the token may have expired
// or it never existed
var ErrTokenNotFound = errors.New("token not found")

// BadRestoreZip provides enough data to produce user-friendly errors
// when an appspace data archive is unusable
type BadRestoreZip interface {
	Error() string
	MissingFiles() []string
	ZipFiles() []string
}
