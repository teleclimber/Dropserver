package nulltypes

import (
	"database/sql"
	"encoding/json"
)

// inspiration:
// - https://medium.com/aubergine-solutions/how-i-handled-null-possible-values-from-database-rows-in-golang-521fb0ee267
// - https://github.com/guregu/null/

// NullString is an alias for mysql.NullTime data type
type NullString struct {
	sql.NullString
}

// NewString creates a new Time.
func NewString(s string, valid bool) NullString {
	var ret NullString
	ret.String = s
	ret.Valid = valid
	return ret
}

// SetString sets valid to tru and sets string field
// TODO: test these...
func (ns *NullString) SetString(str string) {
	ns.Valid = true
	ns.String = str
}

// SetNull sets the Valied filed to fals and zeros the string
func (ns *NullString) SetNull() {
	ns.Valid = false
	ns.String = ""
}

// MarshalJSON for NullString
func (ns NullString) MarshalJSON() ([]byte, error) {
	if !ns.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(ns.String)
}

// ForceString returns the string or empty string if not Valid
func (ns NullString) ForceString() string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
