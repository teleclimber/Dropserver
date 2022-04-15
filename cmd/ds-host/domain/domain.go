package domain

//go:generate mockgen -destination=mocks.go -package=domain -self_package=github.com/teleclimber/DropServer/cmd/ds-host/domain github.com/teleclimber/DropServer/cmd/ds-host/domain MetricsI,SandboxI,V0RouteModel,AppspaceRouteModels,StdInput
// ^^ remember to add new interfaces to list of interfaces to mock ^^

import (
	"errors"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/twine-go/twine"
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
// This is for server-side use only.
type RuntimeConfig struct {
	DataDir string `json:"data-dir"`
	Server  struct {
		Port    int16  `json:"port"`
		Host    string `json:"host"`
		SslCert string `json:"ssl-cert"`
		SslKey  string `json:"ssl-key"`
	} `json:"server"`
	// NoTLS indicates that this instance should be accessed from the outside without TLS
	NoTLS bool `json:"no-tls"`
	// PortString sets the port that will be appended to domains pointing to your instance.
	// If your instance is exposed to the outside world on a non-standard port,
	// use this setting to ensure generated links are correct.
	// Example: ":5050"
	PortString string `json:"port-string"`
	TrustCert  string `json:"trust-cert"`
	Subdomains struct {
		UserAccounts string `json:"user-accounts"`
		StaticAssets string `json:"static-assets"` // this can't just be a subdomain, has to be full domain, (but you could use a cname in DNS, right)
	} `json:"subdomains"`
	Sandbox struct {
		Num         int    `json:"num"`
		SocketsDir  string `json:"sockets-dir"` // do we really need this? could we not put it in DataDir/sockets?
		UseCGroups  bool   `json:"use-cgroups"`
		CGroupMount string `json:"cgroup-mount"`
		MemoryHigh  int    `json:"memory-high"`
	} `json:"sandbox"`
	Log        string `json:"log"`
	Prometheus struct {
		Enable bool   `json:"enable"`
		Port   uint16 `json:"port"`
	} `json:"prometheus"`
	// Exec contains values determined at runtime
	// These are not settable via json.
	Exec struct {
		UserRoutesDomain string
		SandboxCodePath  string
		AppsPath         string
		AppspacesPath    string
	}
}

// APIVersion is the Dropserver API version that a dropserver app interacts with
type APIVersion int

// DB is the global host database handler
// OK, but it does not need to be wrapped in a struct!
type DB struct {
	Handle *sqlx.DB
}

// ErrorCode represents integer codes for each error mesage
type ErrorCode int

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

// MetricsI represents the global Metrics interface
type MetricsI interface {
	HostHandleReq(start time.Time)
}

// SandboxRunIDs contains all the identifiers relevant to a sandbox run
type SandboxRunIDs struct {
	SandboxID  int            `db:"sandbox_id" json:"sandbox_id"`
	Instance   string         `db:"instance" json:"instance"`
	LocalID    int            `db:"local_id" json:"local_id"`
	OwnerID    UserID         `db:"owner_id" json:"owner_id"`
	AppID      AppID          `db:"app_id" json:"app_id"`
	Version    Version        `db:"version" json:"version"`
	AppspaceID NullAppspaceID `db:"appspace_id" json:"appspace_id"`
	Operation  string         `db:"operation" json:"operation"`
	CGroup     string         `db:"cgroup" json:"cgroup"`
}

// SandboxRunData contains the metrics of a sandbox run
type SandboxRunData struct {
	Start       time.Time          `db:"start" json:"start"`
	End         nulltypes.NullTime `db:"end" json:"end"`
	TiedUpMs    int                `db:"tied_up_ms" json:"tied_up_ms"`
	CpuUsec     int                `db:"cpu_usec" json:"cpu_usec"`
	MemoryBytes int                `db:"memory_bytes" json:"memory_bytes"`
}

type SandboxRun struct {
	SandboxRunIDs
	SandboxRunData
}

// Aggreagte data for usage
// is probably composite of structs from various usage sources (cgroups etc...)
type SandboxRunSums struct {
	TiedUpMs     int `db:"tied_up_ms" json:"tied_up_ms"`
	CpuUsec      int `db:"cpu_usec" json:"cpu_usec"`
	MemoryByteMs int `db:"memory_byte_ms" json:"memory_byte_ms"`
}

// SandboxStatus represents the Status of a Sandbox
type SandboxStatus int

const (
	// SandboxPrepared is the initial status
	SandboxPrepared SandboxStatus = iota + 1
	// SandboxStarting sb is starting not ready yet
	SandboxStarting
	// SandboxReady means it's ready to take incoming requests
	SandboxReady
	// SandboxKilling means the system considers it is going down
	SandboxKilling
	// SandboxDead means the PID is dead
	SandboxDead
	// SandboxCleanedUp means metrics have been collected and traces of sandbox removed.
	SandboxCleanedUp
)

// SandboxI describes the interface to a sandbox
type SandboxI interface {
	OwnerID() UserID
	Operation() string
	AppspaceID() NullAppspaceID
	AppVersion() *AppVersion
	ExecFn(AppspaceRouteHandler) error
	SendMessage(int, int, []byte) (twine.SentMessageI, error)
	GetTransport() http.RoundTripper
	TiedUp() bool
	LastActive() time.Time
	NewTask() chan struct{}
	Status() SandboxStatus
	WaitFor(SandboxStatus)
	Start()
	Graceful()
	Kill()
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

// Authentication provides the authenticated data for a request
// OK, but this is confusing when you have auth for user admin stuff and auth for appspaces
// Proably need to separate those out, along with separate cookie tables, etc...
// Thinkn about the meaning of authentication...
// You're either a ds-host user (user id),
// or an appspace user proxy id
// or you have an api key
//...which is all fine, but it means cookies have to be tweaked as well
type Authentication struct {
	Authenticated bool
	UserID        UserID
	AppspaceID    AppspaceID
	ProxyID       ProxyID // for appspace users (including owner)
	UserAccount   bool    // Tells whether this is for user account auth. Otherwise it's for appspace
	CookieID      string  // if there is a cookie
}

// ^^ this should probably be separated into two:
// - UserAccountAuth
// - AppspaceUserAuth

type TimedToken struct {
	Token   string
	Created time.Time
}

//V0AppspaceLoginToken carries user auth data corresponding to a login token
type V0AppspaceLoginToken struct {
	AppspaceID AppspaceID
	DropID     string
	ProxyID    ProxyID
	LoginToken TimedToken
}

// V0LoginTokenRequest is sent to the host that manages the appspace
type V0LoginTokenRequest struct {
	DropID string `json:"dropid"`
	Ref    string `json:"ref"`
}

// V0PostLoginToken is the data sent from appspace host to dropid host
// when a user requests a token to remote log-in
type V0LoginTokenResponse struct {
	Appspace string `json:"appspace"`
	Token    string `json:"token"`
	Ref      string `json:"ref"`
}

// LoginViewData is used to pass messages and parameters to the login page
type LoginViewData struct {
	Message string
	Email   string
}

// SignupViewData is used to pass messages and parameters to the login page
type SignupViewData struct {
	RegistrationOpen bool
	// username?
	Message string
	Email   string
}

///////////////////////////////////
// Data Models:

// Settings represents admin-settable parameters
type Settings struct {
	RegistrationOpen bool `json:"registration_open" db:"registration_open"` //may not need json here?
}

// UserID represents the user ID
type UserID uint32

// AppID is an application ID
type AppID uint32

// Version is a version string like 0.0.1
type Version string

// AppspaceID is a unique ID for an appspace
type AppspaceID uint32

// ContactID is an ID for a user's contact
type ContactID uint32

// ProxyID is an appspace user's id as seen from the appspace
type ProxyID string

// User is basic representation of a DropServer User
type User struct {
	UserID UserID `db:"user_id"`
	Email  string
}

// Cookie represents the server-side representation of a stored cookie
// Might be called DBCookie to differentiate from thing that came from client?
type Cookie struct {
	CookieID string    `db:"cookie_id"`
	Expires  time.Time `db:"expires"`

	// UserID is confusing. is it contact id in case of appspace?
	// what is it for owner of appspace?
	UserID UserID `db:"user_id"`

	// UserAccount indicates whether this cookie is for the user's account management
	UserAccount bool `db:"user_account"`

	// AppspaceID is the appspace that the cookie can authorize
	// It's mutually exclusive with UserAccount.
	AppspaceID AppspaceID `db:"appspace_id"`

	// ProxyID is for appspace users (including owner id)
	ProxyID ProxyID `db:"proxy_id"`

	// DomainName is the domain that the cookie is set to
	// kind of redundant but simplifies sending cookie with updated expiration
	DomainName string `db:"domain"`
}

// DomainData tells how a domain name can be used
type DomainData struct {
	DomainName                string `json:"domain_name"`
	UserOwned                 bool   `json:"user_owned"`
	ForAppspace               bool   `json:"for_appspace"`
	AppspaceSubdomainRequired bool   `json:"appspace_subdomain_required"`
	ForDropID                 bool   `json:"for_dropid"`
	DropIDSubdomainAllowed    bool   `json:"dropid_subdomain_allowed"`
	// DropIDHandleRequired
}

// DomainCheckResult is the result of checking for a domain
// and subdomain's availability and validity for a particular purpose.
type DomainCheckResult struct {
	Valid     bool   `json:"valid"`
	Available bool   `json:"available"`
	Message   string `json:"message"`
}

// UserInvitation represents an invitation for a user to join the DropServer instance
type UserInvitation struct {
	Email string `db:"email" json:"email"`
}

// App represents the data structure for an App.
type App struct {
	OwnerID UserID `db:"owner_id"`
	AppID   AppID  `db:"app_id"`
	Name    string
	Created time.Time
}

// AppVersion represents a set of app files with a version
type AppVersion struct {
	AppID       AppID  `db:"app_id"`
	AppName     string `db:"app_name"`
	Version     Version
	APIVersion  APIVersion `db:"api"`
	Schema      int        `db:"schema"` // that is the schema for the app's own data
	Created     time.Time
	LocationKey string `db:"location_key"`
}

type AppGetKey string

// AppGetMeta has app version data and any errors found in it
type AppGetMeta struct {
	Key             AppGetKey        `json:"key"`
	Schema          int              `json:"schema"`
	PrevVersion     Version          `json:"prev_version"`
	NextVersion     Version          `json:"next_version"`
	Errors          []string         `json:"errors"`
	VersionMetadata AppFilesMetadata `json:"version_metadata,omitempty"`
}

// AppGetEvent contains updates to an app getter process
type AppGetEvent struct {
	Key   AppGetKey `json:"key"`
	Done  bool      `json:"done"`
	Error bool      `json:"error"`
	Step  string    `json:"step"`
}

// Appspace represents the data structure for App spaces.
type Appspace struct {
	OwnerID     UserID     `db:"owner_id"`
	AppspaceID  AppspaceID `db:"appspace_id"`
	AppID       AppID      `db:"app_id"`
	AppVersion  Version    `db:"app_version"`
	DropID      string     `db:"dropid"`
	DomainName  string     `db:"domain_name"`
	Created     time.Time  `db:"created"`
	Paused      bool       `db:"paused"`
	LocationKey string     `db:"location_key"`

	// Config AppspaceConfig ..this one is harder
}

type RemoteAppspace struct {
	UserID      UserID    `db:"user_id"`
	DomainName  string    `db:"domain_name"`
	OwnerDropID string    `db:"owner_dropid"`
	UserDropID  string    `db:"dropid"`
	Created     time.Time `db:"created"`
}

// AppspaceMetaInfo is stored in the appspace meta db
// and represents the current state of the appspace's data
type AppspaceMetaInfo struct {
	Schema int
	// Add more stuff like DS API, and maybe make it versioned later
}

// AppspaceUserPermission describes a permission that can be granted to
// a user, or via other means. The name and description are user-facing,
// the key is used internally.
type AppspaceUserPermission struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AppFilesMetadata containes metadata that can be gleaned from
// reading the application files
// This is going to evolve until it becomes data that comes from the sandbox.
type AppFilesMetadata struct {
	AppName         string                   `json:"name"`
	AppVersion      Version                  `json:"version"`
	APIVersion      APIVersion               `json:"api_version"`
	UserPermissions []AppspaceUserPermission `json:"user_permissions"` // this should be removed.
}

// ErrAppConfigNotFound means the application config (dropapp.json) file was not found
var ErrAppConfigNotFound = errors.New("App config json not found")

// V0AppspaceDBQuery is the structure expected when Posting a DB request
type V0AppspaceDBQuery struct {
	DBName      string                 `json:"db_name"`
	Type        string                 `json:"type"` // "query" or "exec"
	SQL         string                 `json:"sql"`
	Params      []interface{}          `json:"params"`
	NamedParams map[string]interface{} `json:"named_params"`
}

// MigrationJobStatus represents the Status of an appspace's migration to a different version
// including possibly a different schema
type MigrationJobStatus int

const ( //maybe at MigrationWaiting at some point
	// MigrationStarted means the job has started
	MigrationStarted MigrationJobStatus = iota + 1
	// MigrationRunning means the migration sandbox is running and migrating schemas
	MigrationRunning
	// MigrationFinished means the migration is complete or ended with an error
	MigrationFinished
	// When changing cases make sure to also change in response types and in frontend code!
)

// JobID is the id of appspace migration job
type JobID int

// MigrationJob describes a pending or ongoing appspace migration job
type MigrationJob struct {
	JobID      JobID                `db:"job_id" json:"job_id"`
	OwnerID    UserID               `db:"owner_id" json:"owner_id"`
	AppspaceID AppspaceID           `db:"appspace_id" json:"appspace_id"`
	ToVersion  Version              `db:"to_version" json:"to_version"`
	Created    time.Time            `db:"created" json:"created"`
	Started    nulltypes.NullTime   `db:"started" json:"started"`
	Finished   nulltypes.NullTime   `db:"finished" json:"finished"`
	Priority   bool                 `db:"priority" json:"priority"`
	Error      nulltypes.NullString `db:"error" json:"error"`
}

// AppspaceRouteHandler is a JSON friendly struct
// that describes the desired handling for the route
type AppspaceRouteHandler struct {
	Type     string `json:"type"`           // how can we validate that "type" is entered corrently?
	File     string `json:"file,omitempty"` // this is called "location" downstream. (but why?)
	Function string `json:"function,omitempty"`
	Path     string `json:"path,omitempty"`
}

// AppspaceRouteAuth is a JSON friendly struct that contains
// description of auth paradigm for a route
// Will need a lot more than just type in the long run.
type AppspaceRouteAuth struct {
	//Allow is either "public", "authorized"
	Allow string `json:"allow"`
	// Permission that is required to access this route
	// An empty string means no specifc permission needed
	Permission string `json:"permission"`
}

// AppspaceRouteConfig gives necessary data to handle an appspace route
type AppspaceRouteConfig struct {
	Methods []string             `json:"methods"`
	Path    string               `json:"path"`
	Auth    AppspaceRouteAuth    `json:"auth"`
	Handler AppspaceRouteHandler `json:"handler"`
}

// migration data comign from sandbox:
// this should  also include description as string.
// Maybe an explicit "lossy" flag? Or Lossless so taht default false doesn't imply things that aren't true.
type MigrationStep struct {
	Direction string `json:"direction"`
	Schema    int    `json:"schema"`
}

// New appspace route stuff:

// V0AppRoute is route config for appspace as stored with app
type V0AppRoute struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    V0AppRoutePath    `json:"path"` // Path is the request path to match
	Auth    AppspaceRouteAuth `json:"auth"`
	Type    string            `json:"type"`    // Type of handler: "function" or "static" for now
	Options V0AppRouteOptions `json:"options"` // Options for the route handler
}

// V0AppRoutePath is the request path twe are seeking to match
type V0AppRoutePath struct {
	Path string `json:"path"`
	End  bool   `json:"end"` // End of false makes the path a wildcard /path/**
}

// V0AppRouteOptions is a JSON friendly struct
// that describes the desired handling for the route
type V0AppRouteOptions struct {
	Name string `json:"name,omitempty"` // this is called "location" downstream. (but why?)
	Path string `json:"path,omitempty"`
}

// DropID represents a golbally unique identification
// a user can ue to communicate with other DropServer instances
type DropID struct {
	UserID      UserID    `db:"user_id" json:"user_id"`
	Handle      string    `db:"handle" json:"handle" validate:"nonzero,max=100"`
	Domain      string    `db:"domain" json:"domain" validate:"nonzero,max=100"`
	DisplayName string    `db:"display_name" json:"display_name" validate:"max=100"`
	Created     time.Time `db:"created" json:"created_dt"`
}

// Contact represents a user's contact
// Q: where how when do we attach other props lige appspace use and auth methods?
type Contact struct {
	UserID      UserID    `db:"user_id"`
	ContactID   ContactID `db:"contact_id"`
	Name        string    `db:"name"`
	DisplayName string    `db:"display_name"`
	Created     time.Time `db:"created"`
}

// AppspaceUser identifies a user of an appspace
// Not sure we want this to have this form? Auth should be its own struct?
// TODO this is AppspaceUserV0
type AppspaceUser struct {
	AppspaceID  AppspaceID         `json:"appspace_id"`
	ProxyID     ProxyID            `json:"proxy_id"`
	AuthType    string             `json:"auth_type"`
	AuthID      string             `json:"auth_id"`
	DisplayName string             `json:"display_name"`
	Avatar      string             `json:"avatar"`
	Permissions []string           `json:"permissions"`
	Created     time.Time          `json:"created_dt"`
	LastSeen    nulltypes.NullTime `json:"last_seen"`
}

// V0RouteModel serves route data queries at version 0
type V0RouteModel interface {
	ReverseServiceI

	Create(methods []string, url string, auth AppspaceRouteAuth, handler AppspaceRouteHandler) error

	// Get returns all routes that
	// - match one of the methods passed, and
	// - matches the routePath exactly (no interpolation is done to match sub-paths)
	Get(methods []string, routePath string) (*[]AppspaceRouteConfig, error)
	GetAll() (*[]AppspaceRouteConfig, error)
	GetPath(string) (*[]AppspaceRouteConfig, error)

	Delete(methods []string, url string) error

	// Match finds the route that should handle the request
	// The path will be broken into parts to find the subset path that matches.
	// It returns (nil, nil) if no matches found
	Match(method string, url string) (*AppspaceRouteConfig, error)
}

// AppspaceRouteModels returns models of the desired version
type AppspaceRouteModels interface {
	GetV0(AppspaceID) V0RouteModel
}

// ReverseServiceI is a common interface for reverse services of all versions
type ReverseServiceI interface {
	HandleMessage(twine.ReceivedMessageI)
}

// SandboxMigrateService is the Twine service ID for appspace migration
const SandboxMigrateService = 13

// SandboxAppService is the Twine service ID for querying the app for things like routes, exec fns,
// ..and anything else that is set in code.
const SandboxAppService = 14

// TwineService attach to a twine instance to handle two-way communication
// with a remote twine client
type TwineService interface {
	Start(UserID, *twine.Twine)
	HandleMessage(twine.ReceivedMessageI)
}

// TwineService2 returns a TwineServiceI on start
type TwineService2 interface {
	Start(UserID, *twine.Twine) TwineServiceI
}

// TwineServiceI handles incoming messages for a twine connection
type TwineServiceI interface {
	HandleMessage(twine.ReceivedMessageI)
}

// Events...

//AppspacePausedEvent is the payload for appspace paused event
type AppspacePausedEvent struct {
	AppspaceID AppspaceID
	Paused     bool
}

// AppspaceRouteEvent carries information about a change in an appspace's routes
// type AppspaceRouteEvent struct {
// 	AppspaceID AppspaceID `json:"appspace_id"`
// 	Path       string     `json:"path"`
// }

//AppspaceStatusEvent indicates readiness of appspace and the reason
type AppspaceStatusEvent struct {
	AppspaceID       AppspaceID `json:"appspace_id"`
	Paused           bool       `json:"paused"`
	TempPaused       bool       `json:"temp_paused"`
	TempPauseReason  string     `json:"temp_pause_reason"`
	AppspaceSchema   int        `json:"appspace_schema"`
	AppVersionSchema int        `json:"app_version_schema"`
	Problem          bool       `json:"problem"` // string? To hint at the problem?
}

// LoggerI is interface for appspace and app log
type LoggerI interface {
	Log(source, message string)
	SubscribeStatus() (bool, <-chan bool)
	UnsubscribeStatus(ch <-chan bool)
	GetLastBytes(n int64) (LogChunk, error)
	SubscribeEntries(n int64) (LogChunk, <-chan string, error)
	UnsubscribeEntries(ch <-chan string)
}

// AppspaceLogChunk contains a part of an appspace Log as a string
// and the from and to bytes that this string represents in the log
type LogChunk struct {
	From    int64  `json:"from"`
	To      int64  `json:"to"`
	Content string `json:"content"`
}

// AppspaceRouteHitEvent contains the route that was matched with the request
// Is this versioned or not? It would be easier if not.
// Or at least have basic data unversioned, and more details versioned?
type AppspaceRouteHitEvent struct {
	Timestamp     time.Time
	AppspaceID    AppspaceID
	Request       *http.Request
	V0RouteConfig *V0AppRoute // use generic app route, not versioned
	// Credentials presented by the requester
	// zero-values indicate credential not presented
	Credentials struct {
		ProxyID ProxyID
	}
	// Authorized: whether the route was authorized or not
	Authorized bool
	Status     int
}

// cli stuff

// StdInput gives ability to read from the command line
type StdInput interface {
	ReadLine(string) string
}
