package nulltypes

import (
	"encoding/json"
	"testing"
)

var ()

// not sure how to test this.
func TestMarshalString(t *testing.T) {
	ts := NewString("abc", true)

	data, err := json.Marshal(ts)
	if err != nil {
		t.Error(err)
	}
	assertJSONEquals(t, data, "\"abc\"", "non-empty json marshal")

	ts.Valid = false
	data, err = json.Marshal(ts)
	if err != nil {
		t.Error(err)
	}
	assertJSONEquals(t, data, "null", "null json marshal")
}
