package domain

import (
	"database/sql"
	"encoding/json"
)

// inspiration:
// - https://medium.com/aubergine-solutions/how-i-handled-null-possible-values-from-database-rows-in-golang-521fb0ee267
// - https://github.com/guregu/null/

// NullString is an alias for mysql.NullTime data type
type NullAppspaceID struct {
	sql.NullInt32
}

// NewString creates a new Time.
func NewNullAppspaceID(a ...AppspaceID) (ret NullAppspaceID) {
	if len(a) != 0 {
		ret.Valid = true
		ret.Int32 = int32(a[0])
	}
	return
}

// SetString sets valid to tru and sets string field
func (n *NullAppspaceID) Set(a AppspaceID) {
	n.Valid = true
	n.Int32 = int32(a)
}

// SetNull sets the Valied filed to fals and zeros the string
func (n *NullAppspaceID) Unset() {
	n.Valid = false
	n.Int32 = 0
}

func (n *NullAppspaceID) Get() (AppspaceID, bool) {
	return AppspaceID(n.Int32), n.Valid
}

func (n NullAppspaceID) Equal(c NullAppspaceID) bool {
	if !n.Valid && !c.Valid {
		return true
	}
	if n.Valid != c.Valid {
		return false
	}
	return n.Int32 == c.Int32
}

// MarshalJSON for NullAppspaceID
func (n *NullAppspaceID) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(n.Int32)
}
