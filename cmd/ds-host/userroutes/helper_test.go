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

	err = readJSON(req, d)
	if err != nil {
		t.Fatal(err)
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

func TestWriteEmptyArray(t *testing.T) {
	rr := httptest.NewRecorder()

	d := struct {
		Arr []testData `json:"arr"`
	}{}

	writeJSON(rr, d)

	if rr.Code != http.StatusOK {
		t.Fatal("got bad status code")
	}

	bodyStr := rr.Body.String()

	if !strings.Contains(bodyStr, `"arr":null`) {
		t.Fatal("body not as expected" + bodyStr)
	}

	// empty arrays end up being null in Golang's Marshal, sorry.
	// https://github.com/golang/go/issues/27589
}
