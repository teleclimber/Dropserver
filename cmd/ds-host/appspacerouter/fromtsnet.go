package appspacerouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type FromTSNet struct {
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceUserModel interface {
		GetByAuth(appspaceID domain.AppspaceID, authType string, identifier string) (domain.AppspaceUser, error)
	} `checkinject:"required"`
	AppspaceRouter interface {
		BuildRoutes(mux *chi.Mux)
	} `checkinject:"required"`

	mux *chi.Mux
}

func (f *FromTSNet) Init() {
	f.mux = chi.NewRouter()

	// the appspacetsnet server will load the appspace, since it has it right there.
	// mux.Use(f.loadAppspace)

	// appspacetsnet server will put the ts id in context
	// But we need to look up a proxy id if there is one
	// Then the big questions become how to handle users?
	f.mux.Use(f.getProxyID)

	//after that need to pass on to the actual appspace router.
	f.AppspaceRouter.BuildRoutes(f.mux)
}

func (f *FromTSNet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mux.ServeHTTP(w, r)
}

func (f *FromTSNet) getProxyID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)
		tsUserID, ok := domain.CtxTSNetUserID(ctx)
		if !ok {
			f.getLogger(appspace.AppspaceID).Debug("getProxyID() no tsnet user id")
			next.ServeHTTP(w, r)
		}

		f.getLogger(appspace.AppspaceID).Debug("tsnet user id: " + tsUserID)

		u, err := f.AppspaceUserModel.GetByAuth(appspace.AppspaceID, "tsid", tsUserID)
		if err == domain.ErrNoRowsInResultSet {
			f.getLogger(appspace.AppspaceID).Debug("getProxyID() no sql rows for tsid")
		}
		if err != nil {
			f.getLogger(appspace.AppspaceID).AddNote("AppspaceUserModel.GetByAuth").Error(err)
		}

		next.ServeHTTP(w, r.WithContext(domain.CtxWithAppspaceUserProxyID(ctx, u.ProxyID)))
	})
}

func (f *FromTSNet) getLogger(appspaceID domain.AppspaceID) *record.DsLogger {
	return record.NewDsLogger().AppspaceID(appspaceID).AddNote("FromTSNet")
}
