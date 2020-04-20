package appspaceroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// handles /dropserver/ routes of an app-space
// one of these is token-based authentication

// This should be subject to DS API version.

// DropserverRoutesV0 represents struct for /dropserver appspace routes
type DropserverRoutesV0 struct {
}

func (r *DropserverRoutesV0) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	http.Error(res, "not implemented", http.StatusNotImplemented)
}
