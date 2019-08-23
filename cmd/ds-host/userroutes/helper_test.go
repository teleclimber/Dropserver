package userroutes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testData struct {
	Foo string `json:"bar"`
}

func TestReadJson(t *testing.T) {
	jsonStr := []byte(`{"bar":"lol"}`)
	req, err := http.NewRequest("GET", "/", bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	d := &testData{}

	dsErr := readJSON(req, d)
	if dsErr != nil {
		t.Fatal(dsErr)
	}

	if d.Foo != "lol" {
		t.Fatal("was supposed to get Foo === lol")
	}

}

func TestWriteJSON(t *testing.T) {
	rr := httptest.NewRecorder()

	d := testData{
		Foo: "lol"}

	writeJSON(rr, d)

	if rr.Code != http.StatusOK {
		t.Fatal("got bad status code")
	}

	bodyStr := rr.Body.String()

	if !strings.Contains(bodyStr, `"bar":"lol"`) {
		t.Fatal("body not as expected" + bodyStr)
	}
}
