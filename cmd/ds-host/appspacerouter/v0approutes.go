package appspacerouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	pathToRegexp "github.com/soongo/path-to-regexp"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
)

// Here we load and stash the
//

type compiledRoute struct {
	domain.V0AppRoute
	match func(string) (*pathToRegexp.MatchResult, error)
}

type appVersionKey struct {
	appID      domain.AppID
	appVersion domain.Version
}

type V0AppRoutes struct {
	AppModel interface {
		GetVersion(appID domain.AppID, version domain.Version) (domain.AppVersion, error)
	} `checkinject:"required"`
	AppFilesModel interface {
		ReadRoutes(locationKey string) ([]byte, error)
	} `checkinject:"required"`
	Config *domain.RuntimeConfig `checkinject:"required"`

	appRoutes map[appVersionKey][]compiledRoute

	// also track last usage timestamp so flush unused
}

func (r *V0AppRoutes) Init() {
	r.appRoutes = make(map[appVersionKey][]compiledRoute)
}

func (r *V0AppRoutes) Match(appID domain.AppID, version domain.Version, method string, reqPath string) (domain.V0AppRoute, error) {
	k := appVersionKey{appID, version}
	routes, err := r.load(k)
	if err != nil {
		return domain.V0AppRoute{}, err
	}

	method = strings.ToLower(method)

	for _, route := range routes {
		if strings.ToLower(route.Method) != method {
			continue
		}
		m, err := route.match(reqPath)
		if err != nil {
			r.getLogger("Match").Error(err)
			return domain.V0AppRoute{}, err
		}
		if m != nil {
			return route.V0AppRoute, nil
		}
	}
	return domain.V0AppRoute{}, nil
}

func (r *V0AppRoutes) load(k appVersionKey) ([]compiledRoute, error) {
	routes, ok := r.appRoutes[k]
	if ok {
		return routes, nil
	}

	appVersion, err := r.AppModel.GetVersion(k.appID, k.appVersion)
	if err != nil {
		return []compiledRoute{}, err
	}

	routesData, err := r.AppFilesModel.ReadRoutes(appVersion.LocationKey)
	if err != nil {
		return []compiledRoute{}, err
	}

	var stored []domain.V0AppRoute
	err = json.Unmarshal(routesData, &stored)
	if err != nil {
		r.getLogger("load, json.Unmarshal").Error(err)
		return []compiledRoute{}, err
	}

	return r.compile(stored)
}
func (r *V0AppRoutes) compile(storedRoutes []domain.V0AppRoute) ([]compiledRoute, error) {

	compiled := make([]compiledRoute, len(storedRoutes))

	for i, s := range storedRoutes {
		match, err := r.getMatchFn(s)
		if err != nil {
			r.getLogger("load, pathToRegexp").AddNote(s.ID).Error(err)
			return []compiledRoute{}, err
		}
		compiled[i] = compiledRoute{s, match}
	}

	return compiled, nil
}

// ValidateRoutes validates routes passed to it
func (r *V0AppRoutes) ValidateRoutes(routes []domain.V0AppRoute) error {
	if len(routes) == 0 {
		return errors.New("there should be at least one route")
	}

	for _, route := range routes {
		err := r.validateStoredRoute(route)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *V0AppRoutes) validateStoredRoute(route domain.V0AppRoute) error {
	// TODO validate mehod, route type, and Auth!

	if route.Path.Path == "" {
		return errors.New("empty path is not allowed")
	}
	var tokens []pathToRegexp.Token
	options := getOptions(route)
	_, err := pathToRegexp.PathToRegexp(route.Path.Path, &tokens, &options)
	if err != nil {
		// need to return a good erro
		return fmt.Errorf("failed to turn path %v into regex: %w", route.Path, err)
	}
	if route.Type == "static" && len(tokens) != 0 {
		return fmt.Errorf("static route %v can not have path parameters", route.Path)
	}
	// - check permissions required for routes match declared permissions, or is that done elsewhere?

	return nil
}

func (r *V0AppRoutes) getMatchFn(route domain.V0AppRoute) (func(string) (*pathToRegexp.MatchResult, error), error) {
	options := getOptions(route)
	matchFn, err := pathToRegexp.Match(route.Path.Path, &options)
	return matchFn, err
}

func (r *V0AppRoutes) getLogger(note string) *record.DsLogger {
	l := record.NewDsLogger().AddNote("V0AppRoutes")
	if note != "" {
		l.AddNote(note)
	}
	return l
}

func getOptions(route domain.V0AppRoute) pathToRegexp.Options {
	return pathToRegexp.Options{
		End: &route.Path.End,
	}
}
