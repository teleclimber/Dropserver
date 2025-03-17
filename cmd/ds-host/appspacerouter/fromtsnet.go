package appspacerouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type FromTSNet struct {
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
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

	f.mux.Use(f.loadAppspace)

	// appspace TSNetNode server puts the tsnet user id in context
	f.mux.Use(f.getProxyID)

	// after that need to pass on to the actual appspace router.
	f.AppspaceRouter.BuildRoutes(f.mux)
}

func (f *FromTSNet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mux.ServeHTTP(w, r)
}

func (f *FromTSNet) loadAppspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appspaceID, ok := domain.CtxAppspaceID(r.Context())
		if !ok {
			// That's a developer error, so panic
			panic("no appspace ID in context")
		}
		appspace, err := f.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			f.getLogger(appspaceID).AddNote("AppspaceModel.GetFromID").Error(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if appspace == nil {
			f.getLogger(appspaceID).AddNote("AppspaceModel.GetFromID").Log("no appspace returned")
			// this is an internal error because tsnet should have shut down before the appspace got deleted
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))
		next.ServeHTTP(w, r)
	})
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

		u, err := f.AppspaceUserModel.GetByAuth(appspace.AppspaceID, "tsnetid", tsUserID)
		if err == domain.ErrNoRowsInResultSet {
			f.getLogger(appspace.AppspaceID).Debug("getProxyID() no sql rows for tsnetid")
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
