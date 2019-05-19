package domain

//go:generate mockgen -destination=mocks.go -package=domain github.com/teleclimber/DropServer/cmd/ds-host/domain DBManagerI,LogCLientI,MetricsI,SandboxI,SandboxManagerI,RouteHandler,AppModel,AppspaceModel,TrustedClientI
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
		Address string `json:"address"`
	} `json:"loki"`
	Prometheus struct {
		Port int16 `json:"port"`
	} `json:"prometheus"`
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

// TrustedConfig is the runtime configuration data for ds-trusted
type TrustedConfig struct {
	Loki struct {
		Port    int16  `json:"port"`
		Address string `json:"address"`
	} `json:"loki"`
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

// TODO: do for TrustedManager?

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
// -> wait is this not AppspaceRouteHandler?
// Or do we use the sameRouteData? Surely quite a lot is in common?
// ..but would it not muddy the meaning of the Fields?
type RouteHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request, *AppspaceRouteData)
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

// AppModel is the interface for the appspace model
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

// TrustedClientI is the interface for the client of the ds-trusted remote service
type TrustedClientI interface {
	Init(string)
	SaveAppFiles(*TrustedSaveAppFiles) (*TrustedSaveAppFilesReply, Error)
	GetAppMeta(*TrustedGetAppMeta) (*TrustedGetAppMetaReply, Error)
}

// huh, we could almost make trusted client and server the same interface?!?

// TrustedSaveAppFiles is args for trusted RPC call
type TrustedSaveAppFiles struct {
	Files *map[string][]byte
}

// TrustedSaveAppFilesReply is reply
type TrustedSaveAppFilesReply struct {
	LocationKey string
}

// TrustedGetAppMeta is arguments for GetAppMeta
type TrustedGetAppMeta struct {
	LocationKey string
}

// TrustedGetAppMetaReply is the reply contain application metadata.
// May need to contain a domain-wide application meta?
// Or app-file-meta, given there will be app-meta from DB.
// In fact this should just be a app-file meta struct that is returned?
// ..though slightly concerned about the versioning of application meta data.
type TrustedGetAppMetaReply struct {
	AppFilesMetadata AppFilesMetadata
}

// AppFilesMetadata containes metadata that can be gleaned from
// reading the application files
type AppFilesMetadata struct {
	AppName    string  `json:"name"`
	AppVersion Version `json:"version"`
	// there is a whole gaggle of stuff, at least according to earlier node version.
	// currently we have it in app.json what the routes are.
}
