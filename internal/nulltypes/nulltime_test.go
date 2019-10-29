package nulltypes

import (
	"encoding/json"
	"testing"
	"time"
)

var (
	timeString   = "2012-12-21T21:21:21Z"
	timeJSON     = []byte(`"` + timeString + `"`)
	nullTimeJSON = []byte(`null`)
	timeValue, _ = time.Parse(time.RFC3339, timeString)
	timeObject   = []byte(`{"Time":"2012-12-21T21:21:21Z","Valid":true}`)
)

// not sure how to test this.
func TestMarshalTim(t *testing.T) {
	ti := NewTime(timeValue, true)

	data, err := json.Marshal(ti)
	if err != nil {
		t.Error(err)
	}
	assertJSONEquals(t, data, string(timeJSON), "non-empty json marshal")

	ti.Valid = false
	data, err = json.Marshal(ti)
	if err != nil {
		t.Error(err)
	}
	assertJSONEquals(t, data, string(nullTimeJSON), "null json marshal")
}

func assertJSONEquals(t *testing.T, data []byte, cmp string, from string) {
	if string(data) != cmp {
		t.Errorf("bad %s data: %s ≠ %s\n", from, data, cmp)
	}
}
