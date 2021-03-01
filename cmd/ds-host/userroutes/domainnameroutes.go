package userroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type DomainData struct {
	DomainName string `json:"domain_name"`
}

// DomainNameRoutes is currently just functional enough to support creating drop ids
// Ultimately adding domains and setting what they're for is a whole thing that will take place here.
type DomainNameRoutes struct {
	Config *domain.RuntimeConfig
}

func (d *DomainNameRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	switch req.Method {
	case http.MethodGet:
		writeJSON(res, []DomainData{{DomainName: d.Config.Exec.UserRoutesDomain}})
	default:
		returnError(res, errBadRequest)
	}
}
