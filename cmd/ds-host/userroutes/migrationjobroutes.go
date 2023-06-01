package userroutes

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// MigrationJobRoutes can initiate and return appspace migration jobs
type MigrationJobRoutes struct {
	AppModel interface {
		GetVersion(domain.AppID, domain.Version) (domain.AppVersion, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	} `checkinject:"required"`
	MigrationJobModel interface {
		Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error)
		GetJob(jobID domain.JobID) (*domain.MigrationJob, error)
		GetForAppspace(appspaceID domain.AppspaceID) ([]*domain.MigrationJob, error)
	} `checkinject:"required"`
	MigrationJobController interface {
		WakeUp()
	} `checkinject:"required"`
}

func (j *MigrationJobRoutes) subRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(mustBeAuthenticated)

	r.Get("/", j.getJobsQuery)
	r.Post("/", j.postNewJob)

	r.Route("/{job}", func(r chi.Router) {
		r.Get("/", j.getJob)
	})

	return r
}

// PostAppspaceVersionReq is
// Later include the date time for scheduling the job
type PostAppspaceVersionReq struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
	Version    domain.Version    `json:"to_version"` // could include app_id to future proof and to verify apples-apples
}

func (j *MigrationJobRoutes) postNewJob(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	reqData := PostAppspaceVersionReq{}
	err := readJSON(r, &reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// minimally validate version string? At least to see if it's not a huge string that would bog down the DB

	appspace, err := j.AppspaceModel.GetFromID(reqData.AppspaceID)
	if err != nil {
		http.Error(w, "error getting appspace: "+err.Error(), 500)
		return
	}
	if appspace.OwnerID != userID {
		http.Error(w, "", http.StatusForbidden)
		return
	}

	_, err = j.AppModel.GetVersion(appspace.AppID, reqData.Version) // maybe not that necessary. Just let the job bomb out. Nothing prevents (for now) someone from deleting the version after creating the job
	if err != nil {
		http.Error(w, "error getting app version: "+err.Error(), 500)
		return
	}

	job, err := j.MigrationJobModel.Create(userID, appspace.AppspaceID, reqData.Version, true)
	if err != nil {
		http.Error(w, "error creating job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	j.MigrationJobController.WakeUp()

	writeJSON(w, *job)
}

func (j *MigrationJobRoutes) getJobsQuery(w http.ResponseWriter, r *http.Request) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	// assumes there is a query string with job ids (potentially)
	// ..or by appspace id, or by user, or...

	query := r.URL.Query()
	appspaceIDs, ok := query["appspace_id"]
	if ok {
		// assume single appspace id for now.
		appspaceIDInt, err := strconv.Atoi(appspaceIDs[0])
		if err != nil {
			http.Error(w, "failed to parse appspace id", http.StatusBadRequest)
			return
		}
		appspaceID := domain.AppspaceID(appspaceIDInt)

		// Here we have to ensure that appspace id is authorized for user.
		// so have to load appspace, and compare
		appspace, err := j.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if appspace.OwnerID != userID {
			http.Error(w, "", http.StatusForbidden)
			return
		}

		jobs, err := j.MigrationJobModel.GetForAppspace(appspaceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// we could filter on pending/ongoing jobs only here.
		retData := make([]domain.MigrationJob, len(jobs))
		for i, job := range jobs {
			retData[i] = *job
		}

		writeJSON(w, retData)
	}
}

func (j *MigrationJobRoutes) getJob(w http.ResponseWriter, r *http.Request) {
	job, err := j.getJobFromRequest(r)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, job)
}

func (j *MigrationJobRoutes) getJobFromRequest(r *http.Request) (*domain.MigrationJob, error) {
	userID, _ := domain.CtxAuthUserID(r.Context())

	jobIDStr := chi.URLParam(r, "contact")

	jobIDInt, err := strconv.Atoi(jobIDStr)
	if err != nil {
		return nil, errBadRequest
	}
	jobID := domain.JobID(jobIDInt)

	job, err := j.MigrationJobModel.GetJob(jobID)
	if err != nil {
		return nil, errNotFound // maybe not found maybe internal server error
	}
	if job.OwnerID != userID {
		return nil, errForbidden
	}

	return job, nil
}
