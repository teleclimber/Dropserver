package nulltypes

import (
	"database/sql"
	"fmt"
	"time"
)

// inspiration:
// - https://medium.com/aubergine-solutions/how-i-handled-null-possible-values-from-database-rows-in-golang-521fb0ee267
// - https://github.com/guregu/null/

// NullTime is an alias for mysql.NullTime data type
type NullTime struct {
	sql.NullTime
}

// NewTime creates a new Time.
func NewTime(t time.Time, valid bool) NullTime {
	var ret NullTime
	ret.Time = t
	ret.Valid = valid
	return ret
}

// MarshalJSON for NullTime
func (nt NullTime) MarshalJSON() ([]byte, error) {
	if !nt.Valid {
		return []byte("null"), nil
	}
	val := fmt.Sprintf("\"%s\"", nt.Time.Format(time.RFC3339))
	return []byte(val), nil
}

////
