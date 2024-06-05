package appspacerouter

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type FromTsnet struct {
	AppspaceModel interface {
		GetFromDomain(string) (*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceUserModel interface {
		GetByDropID(appspaceID domain.AppspaceID, dropID string) (domain.AppspaceUser, error)
	} `checkinject:"required"`
	AppspaceRouter interface {
		BuildRoutes(mux *chi.Mux)
	} `checkinject:"required"`

	mux *chi.Mux
}

func (f *FromTsnet) Init() {
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

func (f *FromTsnet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mux.ServeHTTP(w, r)
}

func (f *FromTsnet) getProxyID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		appspace, _ := domain.CtxAppspaceData(ctx)
		_, ok := domain.CtxTsnetUserID(ctx)
		if !ok {
			f.getLogger(appspace.AppspaceID).Debug("getProxyID() no tsnet user id")
			next.ServeHTTP(w, r)
		}

		// Here we need to match tsnet user id with proxy ID.
		// TODO this is TBD.
		// for now temporaroly we'll just use the appspac owner.

		u, err := f.AppspaceUserModel.GetByDropID(appspace.AppspaceID, appspace.DropID)
		if err == sql.ErrNoRows {
			f.getLogger(appspace.AppspaceID).Debug("getProxyID() no sql rows for dropid")
		}
		if err != nil {
			f.getLogger(appspace.AppspaceID).AddNote("AppspaceUserModel.GetByDropID").Error(err)
		}

		next.ServeHTTP(w, r.WithContext(domain.CtxWithAppspaceUserProxyID(ctx, u.ProxyID)))
	})
}

func (f *FromTsnet) getLogger(appspaceID domain.AppspaceID) *record.DsLogger {
	return record.NewDsLogger().AppspaceID(appspaceID).AddNote("FromTsnet")
}
