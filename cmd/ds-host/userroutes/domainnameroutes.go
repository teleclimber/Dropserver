package userroutes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// DomainNameRoutes is currently just functional enough to support creating drop ids
// Ultimately adding domains and setting what they're for is a whole thing that will take place here.
type DomainNameRoutes struct {
	DomainController interface {
		GetDomains(userID domain.UserID) ([]domain.DomainData, error)
		CheckAppspaceDomain(userID domain.UserID, dom string, subdomain string) (domain.DomainCheckResult, error)
	}
}

func (d *DomainNameRoutes) subRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/", d.getAvailableDomains)
	r.Get("/check", d.checkDomain)

	return r
}

func (d *DomainNameRoutes) getAvailableDomains(w http.ResponseWriter, r *http.Request) {
	// Another way to look at it is to send all domains with metadata about their potential use
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		//d.getLogger("getUserData").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}
	ret, err := d.DomainController.GetDomains(userID)
	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, ret)
}

func (d *DomainNameRoutes) checkDomain(w http.ResponseWriter, r *http.Request) {
	userID, ok := domain.CtxAuthUserID(r.Context())
	if !ok {
		//d.getLogger("getUserData").Error(errors.New("no auth user id"))
		httpInternalServerError(w)
		return
	}

	query := r.URL.Query()

	domainNames, ok := query["domain"]
	if !ok || len(domainNames) == 0 {
		returnError(w, errBadRequest)
	}
	domainName := domainNames[0]

	subDomains, ok := query["subdomain"]
	if !ok || len(subDomains) == 0 {
		returnError(w, errBadRequest)
	}
	subDomain := subDomains[0]

	var checkResult domain.DomainCheckResult
	var err error
	if _, forAppspace := query["appspace"]; forAppspace {
		checkResult, err = d.DomainController.CheckAppspaceDomain(userID, domainName, subDomain)
	} else if _, forDropID := query["dropid"]; forDropID {
		http.Error(w, "dropid not implemented", http.StatusNotImplemented)
		return
	} else {
		returnError(w, errBadRequest)
		return
	}

	if err != nil {
		returnError(w, err)
		return
	}

	writeJSON(w, checkResult)
}

// Is this where we check for availability and validity of subdomains/domains
// ..for appspaces and other stuff.

// What are some routes
// - GET domains for user (including user's domains and domains the user can use)
// - GET check if domain (subdomain+domain) is available for appspace, or for dropid
//   ..checks if domain can be used by user
//   ..checks if the subdomain combination is in use already, and what for
//     -> a full domain can be used for both appspace and dropid simultaneously
//   ..also need to look for super or sub-arrangements of the domain

// We need to be careful here because it's possible to imagine the scenario:
// - someone points dropserver.mysite.com to their DS
// - creates appspaces like abc.dropserver.mysite.com
// - Then decides that they want to point their whole site to DS, mysite.com
// - So you can't just look for subdomains from domains that the user controls, because those can vary.
//   ..you have to start at the naked root domain, whatever it is, and work your way down ensureing no overlap:
// EX: for abc.dropserver.mysite.com
// - mysite is root, .com is tld, so start with root+tld: mysite.com
//   .. and what? you're going to get a bunch of hits
//   -> exact match for "mysite.com" for an appspace, means *.mysite.com not allowed?
//     Or did we decide that it's OK to have domain overlap? -
//       -> We'll have to be careful with cookies, but it's probably the direction we want to go ultimately?
//          ..probably OK but I would rule that out for the dropid domain
