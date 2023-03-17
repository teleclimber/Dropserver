package userroutes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

func readSingleQueryParam(r *http.Request, key string) (string, bool) {
	query := r.URL.Query()
	values, ok := query[key]
	if !ok {
		return "", false
	}
	if len(values) != 1 {
		return "", false
	}
	unescapedVal, err := url.QueryUnescape(values[0])
	if err != nil {
		return "", false
	}
	return unescapedVal, true
}

func readJSON(r *http.Request, data interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	respJSON, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respJSON)
}

func httpInternalServerError(w http.ResponseWriter) {
	writeServerError(w)
}

func writeServerError(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func writeOK(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusOK), http.StatusOK)
}

func writeBadRequest(w http.ResponseWriter, valName string, errStr string) {
	http.Error(w, fmt.Sprintf("Bad Request: %s: %s", valName, errStr), http.StatusBadRequest)
}

func writeNotFound(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
