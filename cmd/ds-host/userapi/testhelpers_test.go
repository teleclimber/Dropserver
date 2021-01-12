package userapi

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func apiReq(api *UserJSONAPI, method string, url string, body string) (*http.Response, []byte) {
	bod := strings.NewReader(body)

	req, err := http.NewRequest(method, url, bod)
	if err != nil {
		panic(err)
	}

	rr := httptest.NewRecorder()

	api.ServeHTTP(rr, req)

	resp := rr.Result()

	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return resp, payload
}

func payloadContains(t *testing.T, payload []byte, strs ...string) {
	out := &bytes.Buffer{}
	json.Indent(out, payload, "", "\t")

	jsonStr := out.String()

	e := false
	for _, str := range strs {
		if !strings.Contains(jsonStr, str) {
			e = true
			t.Errorf("JSON payload: expected to find %v in json", str)
		}
	}

	if e {
		t.Log(jsonStr)
	}
}

func payloadNotContains(t *testing.T, payload []byte, strs ...string) {
	out := &bytes.Buffer{}
	json.Indent(out, payload, "", "\t")

	jsonStr := out.String()

	e := false
	for _, str := range strs {
		if strings.Contains(jsonStr, str) {
			e = true
			t.Errorf("JSON payload: DID NOT expect to find %v in json", str)
		}
	}

	if e {
		t.Log(jsonStr)
	}
}
