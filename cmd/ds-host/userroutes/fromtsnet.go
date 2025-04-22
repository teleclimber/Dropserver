package userroutes

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

type FromTSNet struct {
	UserModel interface {
		GetFromTSNet(string) (domain.User, error)
	} `checkinject:"required"`
	UserRoutes interface {
		BuildRoutes(mux *chi.Mux)
	} `checkinject:"required"`

	mux *chi.Mux
}

func (f *FromTSNet) Init() {
	r := chi.NewRouter()
	r.Use(addCSPHeaders)
	r.Use(f.accountUser)
	f.UserRoutes.BuildRoutes(r)
	f.mux = r
}

func (f *FromTSNet) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	f.mux.ServeHTTP(res, req)
}

// AccountUser middleware sets the user id in context
// from the tsnet user data.
func (f *FromTSNet) accountUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		tsUserID, ok := domain.CtxTSNetUserID(ctx)
		if !ok {
			f.getLogger("accountUser").Error(errors.New("accountUser() no tsnet user id"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		u, err := f.UserModel.GetFromTSNet(tsUserID)
		if err == domain.ErrNoRowsInResultSet {
			f.getLogger("accountUser").Debug("GetFromTSNet() no sql rows for tsnetid")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err != nil {
			f.getLogger("accountUser").AddNote("GetFromTSNet").Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx = domain.CtxWithAuthUserID(ctx, u.UserID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (f *FromTSNet) getLogger(note string) *record.DsLogger {
	return record.NewDsLogger().AddNote("userroutes FromTSNet").AddNote(note)
}
