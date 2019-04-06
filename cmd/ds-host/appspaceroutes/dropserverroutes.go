package appspaceroutes

import (
	"net/http"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// handles /dropserver/ routes of an app-space
// one of these is token-based authentication

// DropserverRoutes represents struct for /dropserver appspace routes
type DropserverRoutes struct {
}

func (r *DropserverRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	http.Error(res, "not implemented", http.StatusNotImplemented)
}
