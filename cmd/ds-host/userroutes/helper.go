package userroutes

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
)

func readJSON(req *http.Request, data interface{}) domain.Error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return dserror.FromStandard(err)
	}

	err = json.Unmarshal(body, data)
	if err != nil {
		return dserror.FromStandard(err)
	}

	return nil
}

func writeJSON(res http.ResponseWriter, data interface{}) {
	respJSON, err := json.Marshal(data)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(respJSON)
}
