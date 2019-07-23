package domain

//go:generate mockgen -destination=mocks.go -package=domain github.com/teleclimber/DropServer/cmd/ds-host/domain DBManagerI,LogCLientI,MetricsI,SandboxI,SandboxManagerI,RouteHandler,CookieModel,UserModel,AppFilesModel,AppModel,AppspaceModel,ASRoutesModel,Authenticator,Validator,Views
// ^^ remember to add new interfaces to list of interfaces to mock ^^

import (
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

// don't import anything
// just define domain structs and interfaces

// domain structs are not given any "methods" (they are not receiver for any function)
// .. I think. This is because it would have to be defined in this package, which is not the idea.

// So a domain struct is a common, standard way of passing data about core things of the domain.
// So there would be a domain.User struct, but no u.ChangeEmail()
// ..the change email function is a coll to the UserModel, which creates and oerates on domain.User

// RuntimeConfig represents the variables that can be set at runtime
// Or at least set via config file or cli flags that get read once
// upon starting ds-host.
type RuntimeConfig struct {
	DataDir string `json:"data-dir"` // this might get confusing when we also config trusted volume data
	Server  struct {
		Port int16  `json:"port"`
		Host string `json:"host"`
	} `json:"server"`
	Sandbox struct {
		Num int `json:"num"`
	} `json:"sandbox"`
	Loki struct {
		Port    int16  `json:"port"`
		Address string `json:"address"` // Address or IP? Or does it not matter for Loki?
	} `json:"loki"`
	Prometheus struct {
		Port int16 `json:"port"`
	} `json:"prometheus"`

	// Exec contains values determined at runtime
	// These are not settable via json.
	Exec struct {
		GoTemplatesDir      string
		WebpackTemplatesDir string
		StaticAssetsDir     string
		PublicStaticAddress string
		UserRoutesAddress   string
	}
}

// DB is the global host database handler
// OK, but it does not need to be wrapped in a struct!
type DB struct {
	Handle *sqlx.DB
}

// DBManagerI is Migration interface
type DBManagerI interface {
	Open() (*DB, Error)
	GetHandle() *DB
	GetSchema() string
	SetSchema(string) Error
}

// ErrorCode represents integer codes for each error mesage
type ErrorCode int

// Error is dropserver error type
type Error interface {
	//Error() string
	Code() ErrorCode
	ExtraMessage() string
	PublicString() string
	ToStandard() error
	HTTPError(http.ResponseWriter)
}

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

// Authenticator is an interface
type Authenticator interface {
	SetForAccount(http.ResponseWriter, UserID) Error
	GetForAccount(http.ResponseWriter, *http.Request, *AppspaceRouteData) bool
}

// Views interface
type Views interface {
	PrepareTemplates()
	Login(http.ResponseWriter, LoginViewData)
	Signup(http.ResponseWriter, SignupViewData)
	UserHome(http.ResponseWriter)
}

// LoginViewData is used to pass messages and parameters to the login page
type LoginViewData struct {
	Message string
	Email   string
}

// SignupViewData is used to pass messages and parameters to the login page
type SignupViewData struct {
	RegistrationClosed bool
	// username?
	Message string
	Email   string
}

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
	Cookie      *Cookie
	App         *App
	Appspace    *Appspace
	URLTail     string
	RouteConfig *RouteConfig
	Subdomains  *[]string
}

// RouteHandler is a generic interface for sub route handling.
// we will need to pass context of some sort
// -> wait is this not AppspaceRouteHandler?
// Or do we use the sameRouteData? Surely quite a lot is in common?
// ..but would it not muddy the meaning of the Fields?
type RouteHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *AppspaceRouteData)
}

// Validator is an interface for validation module
type Validator interface {
	Init()
	Email(string) Error
	Password(string) Error
}

///////////////////////////////////
// Data Models:

// UserID represents the user ID
type UserID uint32

// AppID is an application ID
type AppID uint32

// Version is a version string like 0.0.1
type Version string

// AppspaceID is a nique ID for an appspace
type AppspaceID uint32

// User is basic representation of User
type User struct {
	UserID UserID `db:"user_id"`
	Email  string
}

// CookieModel is the interface for storing and retriving cookies
type CookieModel interface {
	PrepareStatements()
	Get(string) (*Cookie, Error)
	Create(Cookie) (string, Error)
	UpdateExpires(string, time.Time) Error
}

// Cookie represents the server-side representation of a stored cookie
// Might be called DBCookie to differentiate from thing that came from client?
type Cookie struct {
	CookieID string    `db:"cookie_id"`
	UserID   UserID    `db:"user_id"`
	Expires  time.Time `db:"expires"`

	// UserAccount indicates whether this cookie is for the user's account management
	UserAccount bool `db:"user_account"`

	// Appspace is the identifier of the appspace that this cookie gives acess to
	// It's mutually exclusive with UserHome.
	AppspaceID AppspaceID `db:"appspace_id"`
}

// UserModel is the interface for user model
type UserModel interface {
	PrepareStatements()
	Create(string, string) (*User, Error)
	GetFromID(UserID) (*User, Error)
	GetFromEmail(string) (*User, Error)
	GetFromEmailPassword(string, string) (*User, Error)
	IsAdmin(UserID) bool
	MakeAdmin(UserID) Error
	DeleteAdmin(UserID) Error
}

// AppFilesModel represents the application's files saved to disk
type AppFilesModel interface {
	Save(*map[string][]byte) (string, Error)
	ReadMeta(string) (*AppFilesMetadata, Error)
}

// App represents the data structure for an App.
type App struct {
	OwnerID UserID `db:"owner_id"` // just int, or can we wrap that in a type?
	AppID   AppID  `db:"app_id"`
	Name    string
	Created time.Time
}

// AppVersion represents a set of app files with a version
type AppVersion struct {
	AppID       AppID `db:"app_id"`
	Version     Version
	Created     time.Time
	LocationKey string `db:"location_key"`
}

// AppModel is the interface for the app model
type AppModel interface {
	GetFromID(AppID) (*App, Error)
	Create(UserID, string) (*App, Error)
	GetVersion(AppID, Version) (*AppVersion, Error)
	CreateVersion(AppID, Version, string) (*AppVersion, Error)
}

// Appspace represents the data structure for App spaces.
type Appspace struct {
	OwnerID    UserID     `db:"owner_id"`
	AppspaceID AppspaceID `db:"appspace_id"`
	AppID      AppID      `db:"app_id"`
	AppVersion Version    `db:"app_version"`
	Subdomain  string
	Created    time.Time
	Paused     bool

	// Config AppspaceConfig ..this one is harder
}

// AppspaceModel is the interface for the appspace model
type AppspaceModel interface {
	GetFromID(AppspaceID) (*Appspace, Error)
	GetFromSubdomain(string) (*Appspace, Error)
	Create(UserID, AppID, Version, string) (*Appspace, Error)
}

// ASRoutesModel is the appspaces routes model interface
type ASRoutesModel interface {
	GetRouteConfig(AppVersion, string, string) (*RouteConfig, Error)
}

// AppFilesMetadata containes metadata that can be gleaned from
// reading the application files
type AppFilesMetadata struct {
	AppName    string      `json:"name"`
	AppVersion Version     `json:"version"`
	Routes     []JSONRoute `json:"routes"`
	// there is a whole gaggle of stuff, at least according to earlier node version.
	// currently we have it in app.json what the routes are.
}

// JSONRoute represents the json-formatted Routes from application.json
type JSONRoute struct {
	Route     string           `json:"route"`
	Method    string           `json:"method"`
	Authorize string           `json:"authorize"`
	Handler   JSONRouteHandler `json:"handler"`
}

// JSONRouteHandler is the handler part of route in JSON
type JSONRouteHandler struct {
	Type     string `json:"type"` // how can we validate that "type" is entered corrently?
	File     string `json:"file"` // this is called "location" downstream.
	Function string `json:"function"`
}

// RouteConfig gives necessary data to handle a appspace route
type RouteConfig struct {
	Type      string // static, crud, exec, [and maybe filter, or auth to allow "middlewares"?]
	Authorize string
	Location  string
	Function  string
}

// RoutePart is a sub path of an appspace route, with possible handlers
type RoutePart struct {
	GET  *RouteConfig
	POST *RouteConfig
	// ..others
	Parts map[string]*RoutePart
}
