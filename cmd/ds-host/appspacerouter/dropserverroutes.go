package appspacerouter

import (
	"context"
	"net/http"

	"github.com/teleclimber/DropServer/internal/shiftpath"
)

type DropserverRoutes struct {
	V0DropServerRoutes http.Handler
}

func (r *DropserverRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	trail := getURLTail(req.Context())

	head, tail := shiftpath.ShiftPath(trail)
	ctx := context.WithValue(req.Context(), urlTailCtxKey, tail)

	switch head {
	case "apiversions":
		http.Error(res, "api check not implemented", http.StatusNotImplemented)
	case "v0":
		r.V0DropServerRoutes.ServeHTTP(res, req.WithContext(ctx))
	default:
		http.Error(res, "not found", http.StatusNotFound)
	}

}
