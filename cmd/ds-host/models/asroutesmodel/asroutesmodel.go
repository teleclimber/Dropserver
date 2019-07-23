package asroutesmodel

import (
	"strings"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/dserror"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// Do we need wildcard paths? 
// ..or at least some way of expressing that /static/ handler is for anything below
// It could be we just keep track of the last valid handler
// ..and that's the one that is used.
// It's elegant because we can have a generic /static/ handler, and something more specific if needed
// But sometimes you want to specify that a handler only applies if the route is exactly that
// ex: "/" to serve homepage, but we don't want homepage for any other path, even if accidentally entered.

// Wonder if we it would benefit DS to parse routes for variables?
// ..it would go well with CRUD routes for sure.

// ASRoutesModel holds the Appspace Routes Model
type ASRoutesModel struct {
	Logger domain.LogCLientI

	// AppFilesModel is used to access the JSON config data where app routes are declared
	// This is a temporary solution to my current lack of proper architecture.
	AppFilesModel domain.AppFilesModel
	
	appRoutes map[domain.AppID]map[domain.Version]*domain.RoutePart
}

// GetRouteConfig returns the route configuration for the path passed
// Rename to RouteHandler?
func (m *ASRoutesModel) GetRouteConfig(appVersion domain.AppVersion, method string, pathStr string) (*domain.RouteConfig, domain.Error) {
	routes, dsErr := m.getAppRoutes(appVersion)
	if dsErr != nil {
		return nil, dsErr
	}

	method = strings.ToLower(method)

	part := routes
	last := getConfig(part, method)
	var ok bool
	for {
		head, tail := shiftpath.ShiftPath(pathStr)
		head = strings.ToLower(head)
		part, ok = part.Parts[head]
		if !ok {
			break
		}
		c := getConfig(part, method)
		if c != nil {
			last = c
		}
		pathStr = tail
	}

	if last == nil {
		return nil, dserror.New(dserror.AppspaceRouteNotFound)
	}
	return last, nil
}
func getConfig(part *domain.RoutePart, method string) *domain.RouteConfig {
	if part == nil {
		return nil
	}
	switch strings.ToLower(method) {
		case "get": return part.GET
		case "post": return part.POST
	}
	return nil
}

func (m *ASRoutesModel) getAppRoutes(appVersion domain.AppVersion) (*domain.RoutePart, domain.Error) {
	appID := appVersion.AppID
	ver := appVersion.Version
	if m.appRoutes == nil || m.appRoutes[appID] == nil || m.appRoutes[appID][ver] == nil {
		dsErr := m.fetchAppConfig(appVersion)
		if dsErr != nil {
			return nil, dsErr
		}
	}

	return m.appRoutes[appID][ver], nil
}

func (m *ASRoutesModel) fetchAppConfig(appVersion domain.AppVersion) domain.Error {
	appMeta, dsErr := m.AppFilesModel.ReadMeta(appVersion.LocationKey)
	if dsErr != nil {
		return dsErr
	}

	jsonRoutes := appMeta.Routes

	rootRoute, dsErr := m.parseRoutes(jsonRoutes)
	if dsErr != nil {
		return dsErr
	}

	if m.appRoutes == nil {
		m.appRoutes = make(map[domain.AppID]map[domain.Version]*domain.RoutePart)
	}
	if _, ok := m.appRoutes[appVersion.AppID]; !ok {
		m.appRoutes[appVersion.AppID] = make(map[domain.Version]*domain.RoutePart)
	}
	m.appRoutes[appVersion.AppID][appVersion.Version] = rootRoute

	return nil
}

func (m *ASRoutesModel) parseRoutes(jsonRoutes []domain.JSONRoute) (*domain.RoutePart, domain.Error) {
	// probably take the json data, that is formatted into a struct that mirrors it
	// and walk over it building our nested struct.

	rootRoute := &domain.RoutePart{
		Parts: make(map[string]*domain.RoutePart) }
	
	for _, jrc := range jsonRoutes {
		r := jrc.Route

		curPart := rootRoute

		for {
			var part *domain.RoutePart
			var ok bool
			head, tail := shiftpath.ShiftPath(r)
			head = strings.ToLower(head)
			part, ok = curPart.Parts[head]
			if !ok {
				if head == "" {
					part = rootRoute
				} else {
					part = &domain.RoutePart{
						Parts: make(map[string]*domain.RoutePart)}
					curPart.Parts[head] = part
				}
			}
			if tail == "/" {
				dsErr := applyConfig(jrc, part)
				if dsErr != nil {
					return nil, dsErr
				}
				break
			}
			r = tail
			curPart = part
		}
	}

	return rootRoute, nil
}

// applies the json config to that RoutePart
func applyConfig(jsonRoute domain.JSONRoute, routePart *domain.RoutePart) domain.Error {
	rc := domain.RouteConfig{
		Type: jsonRoute.Handler.Type,
		Authorize: jsonRoute.Authorize,
		Location: jsonRoute.Handler.File,
		Function: jsonRoute.Handler.Function }

	switch strings.ToLower(jsonRoute.Method) {
	case "get": 
		if routePart.GET != nil {
			return dserror.New(dserror.AppRouteConfigProblem, "This part of the route is already configured: "+jsonRoute.Route)
		}
		routePart.GET = &rc
	case "post":
		if routePart.POST != nil {
			return dserror.New(dserror.AppRouteConfigProblem, "This part of the route is already configured: "+jsonRoute.Route)
		}
		routePart.POST = &rc
	default: return dserror.New(dserror.AppRouteConfigProblem, "Method not implemented: "+jsonRoute.Method)
	}

	// probably check to ensure rc is not already populated?
	// ..because we can only have one config per route,
	// ..unless it's a middleware of course.
	
	return nil
}

