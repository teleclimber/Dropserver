package userroutes

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// MigrationJobRoutes can initiate and return appspace migration jobs
type MigrationJobRoutes struct {
	AppModel interface {
		GetVersion(domain.AppID, domain.Version) (*domain.AppVersion, error)
	}
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	}
	MigrationJobModel interface {
		Create(ownerID domain.UserID, appspaceID domain.AppspaceID, toVersion domain.Version, priority bool) (*domain.MigrationJob, error)
		GetJob(jobID domain.JobID) (*domain.MigrationJob, error)
		GetForAppspace(appspaceID domain.AppspaceID) ([]*domain.MigrationJob, error)
	}
	MigrationJobController interface {
		WakeUp()
	}
}

func (j *MigrationJobRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if routeData.Authentication == nil || !routeData.Authentication.UserAccount {
		// maybe log it? Frankly this should be a panic.
		// It's programmer error pure and simple. Kill this thing.
		res.WriteHeader(http.StatusInternalServerError) // If we reach this point we dun fogged up
		return
	}

	job, err := j.getJobFromPath(routeData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if job == nil {
		switch req.Method {
		case http.MethodGet:
			j.getJobsQuery(res, req, routeData)
		case http.MethodPost:
			j.postNewJob(res, req, routeData)
		default:
			http.Error(res, "bad method for /migrationjob", http.StatusBadRequest)
		}
	} else {
		j.getJob(res, req, routeData, job)
	}
}

// PostAppspaceVersionReq is
// Later include the date time for scheduling the job
type PostAppspaceVersionReq struct {
	AppspaceID domain.AppspaceID `json:"appspace_id"`
	Version    domain.Version    `json:"to_version"` // could include app_id to future proof and to verify apples-apples
}

func (j *MigrationJobRoutes) postNewJob(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	if req.Method != http.MethodPost {
		http.Error(res, "expected POST", http.StatusBadRequest)
		return
	}

	reqData := PostAppspaceVersionReq{}
	err := readJSON(req, &reqData)
	if err != nil {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// minimally validate version string? At least to see if it's not a huge string that would bog down the DB

	appspace, err := j.AppspaceModel.GetFromID(reqData.AppspaceID)
	if err != nil {
		http.Error(res, "error getting appspace: "+err.Error(), 500)
		return
	}
	if appspace.OwnerID != routeData.Authentication.UserID {
		http.Error(res, "", http.StatusForbidden)
		return
	}

	_, err = j.AppModel.GetVersion(appspace.AppID, reqData.Version) // maybe not that necessary. Just let the job bomb out. Nothing prevents (for now) someone from deleting the version after creating the job
	if err != nil {
		http.Error(res, "error getting app version: "+err.Error(), 500)
		return
	}

	job, err := j.MigrationJobModel.Create(routeData.Authentication.UserID, appspace.AppspaceID, reqData.Version, true)
	if err != nil {
		http.Error(res, "error creating job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	j.MigrationJobController.WakeUp()

	writeJSON(res, *job)
}

func (j *MigrationJobRoutes) getJobsQuery(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// assumes there is a query string with job ids (potentially)
	// ..or by appspace id, or by user, or...

	query := req.URL.Query()
	appspaceIDs, ok := query["appspace_id"]
	if ok {
		// assume single appspace id for now.
		appspaceIDInt, err := strconv.Atoi(appspaceIDs[0])
		if err != nil {
			http.Error(res, "failed to parse appspace id", http.StatusBadRequest)
			return
		}
		appspaceID := domain.AppspaceID(appspaceIDInt)

		// Here we have to ensure that appspace id is authorized for user.
		// so have to load appspace, and compare
		appspace, err := j.AppspaceModel.GetFromID(appspaceID)
		if err != nil {
			http.Error(res, err.Error(), 500)
			return
		}
		if appspace.OwnerID != routeData.Authentication.UserID {
			http.Error(res, "", http.StatusForbidden)
			return
		}

		jobs, err := j.MigrationJobModel.GetForAppspace(appspaceID)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

		// we could filter on pending/ongoing jobs only here.
		retData := make([]domain.MigrationJob, len(jobs))
		for i, job := range jobs {
			retData[i] = *job
		}

		writeJSON(res, retData)
	}
}

func (j *MigrationJobRoutes) getJob(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData, job *domain.MigrationJob) {
	panic("get job not yet implemented")
}

func (j *MigrationJobRoutes) getJobFromPath(routeData *domain.AppspaceRouteData) (*domain.MigrationJob, error) {
	jobIDStr, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail

	if jobIDStr == "" {
		return nil, nil
	}

	jobIDInt, err := strconv.Atoi(jobIDStr)
	if err != nil {
		return nil, err
	}
	jobID := domain.JobID(jobIDInt)

	job, err := j.MigrationJobModel.GetJob(jobID)
	if err != nil {
		return nil, err
	}
	if job.OwnerID != routeData.Authentication.UserID {
		return nil, errors.New("unauthorized")
	}

	return job, nil
}
