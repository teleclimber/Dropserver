package domain

//go:generate mockgen -destination=mocks.go -package=domain github.com/teleclimber/DropServer/cmd/ds-host/domain LogCLientI,MetricsI,SandboxI,SandboxManagerI,RouteHandler,AppModel,AppspaceModel
// ^^ remember to add new interfaces to list of interfaces to mock ^^

import (
	"net/http"
	"time"
)

// don't import anything
// just define domain structs and interfaces

// domain structs are not given any "methods" (they are not receiver for any function)
// .. I think. This is because it would have to be defined in this package, which is not the idea.

// So a domain struct is a common, standard way of passing data about core things of the domain.
// So there would be a domain.User struct, but no u.ChangeEmail()
// ..the change email function is a coll to the UserModel, which creates and oerates on domain.User

// LogLevel represents the logging severity level
type LogLevel int

// DEBUG is for debug
const (
	DEBUG LogLevel = iota
	INFO  LogLevel = iota
	WARN  LogLevel = iota
	ERROR LogLevel = iota
	// DISABLE Maximum level, disables sending or printing
	DISABLE LogLevel = iota
)

// LogCLientI represents an interface for logging
type LogCLientI interface {
	NewSandboxLogClient(string) LogCLientI
	Log(LogLevel, map[string]string, string)
}

// MetricsI represents the global Metrics interface
type MetricsI interface {
	HostHandleReq(start time.Time)
}

// SandboxManagerI is an interface that describes sm
type SandboxManagerI interface {
	GetForAppSpace(app string, appSpace string) chan SandboxI
}

// SandboxI describes the interface to a sandbox
type SandboxI interface {
	GetName() string
	GetAddress() string
	GetTransport() http.RoundTripper
	GetLogClient() LogCLientI
	TaskBegin() chan bool
}

// Server describes the web server
// type Server struct {
// 	// logger?
// 	// mux?
// 	SandboxManager SandboxI // this should be an interface
// 	Metrics        MetricsI
// 	HostAppSpace   *map[string]string
// 	AppSpaceApp    *map[string]string
// 	// this is going to get annoying with too many models and other things
// 	// Most of the things will be needed by routes,
// 	//.. so make it so we can build out routes and middleware by composing them,
// 	// ..thereby only injecting what is needed at each step.
// 	// off the top of my head, packages would be:
// 	// -> one for user routes, one for admin, one for login, one for app-space
// 	// How does this translate to domains?
// }
// ^^ unused for now!
// -> it's not really a core piece of data that gets passed between packages.
// ..it's more of an application logic struct.
// we'll see what that means when we start doing composable routes. Will we need server then?

///////////////////////////////////////////////////////////
// route stuff

// AppspaceRouteData represents data for the route being executed
// instead of passing string for path tail, pass whole request context object, with:
// - *App
// - *Appspace
// - *AuthState (or some such thing that summarizes the auth story for this request)
// - *AppspaceRoute (the route metadata, match path, type, function, auth ...)
// - path tail?
// - golang Context thing? We need to read up on that.
type AppspaceRouteData struct {
	App        *App
	Appspace   *Appspace
	URLTail    string
	Subdomains *[]string
}

// RouteHandler is a generic interface for sub route handling.
// we will need to pass context of some sort
type RouteHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *AppspaceRouteData)
}

///////////////////////////////////
// Data Models:

// App represents the data structure for an App.
type App struct {
	Name string
	// Version string
	// OwnerID UserID
}

// AppModel is the interface for the appspace model
type AppModel interface {
	GetForName(string) (*App, bool)
	Create(*App)
}

// Appspace represents the data structure for App spaces.
type Appspace struct {
	Name    string
	AppName string
	// AppVersion string
	// Paused bool
	// OwnerID UserID
	// Config AppspaceConfig ..this one is harder
}

// AppspaceModel is the interface for the appspace model
type AppspaceModel interface {
	GetForName(string) (*Appspace, bool)
	Create(*Appspace)
}
