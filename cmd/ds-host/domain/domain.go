package domain

//go:generate mockgen -destination=mocks.go -package=domain -self_package=github.com/teleclimber/DropServer/cmd/ds-host/domain github.com/teleclimber/DropServer/cmd/ds-host/domain DBManagerI,MetricsI,SandboxI,SandboxManagerI,RouteHandler,CookieModel,SettingsModel,UserModel,UserInvitationModel,Validator,Views,DbConn,AppspaceMetaDB,AppspaceInfoModel,V0RouteModel,AppspaceRouteModels,StdInput
// ^^ remember to add new interfaces to list of interfaces to mock ^^

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/DropServer/internal/twine"
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
		NoSsl   bool   `json:"no-ssl"`
		SslCert string `json:"ssl-cert"`
		SslKey  string `json:"ssl-key"`
	} `json:"server"`
	Subdomains struct {
		UserAccounts string `json:"user-accounts"`
		StaticAssets string `json:"static-assets"` // this can't just be a subdomain, has to be full domain, (but you could use a cname in DNS, right)
	} `json:"subdomains"`
	Sandbox struct {
		Num        int    `json:"num"`
		SocketsDir string `json:"sockets-dir"` // do we really need this? could we not put it in DataDir/sockets?
	} `json:"sandbox"`
	Prometheus struct {
		Port int16 `json:"port"`
	} `json:"prometheus"`

	// Exec contains values determined at runtime
	// These are not settable via json.
	Exec struct {
		GoTemplatesDir      string
		WebpackTemplatesDir string
		StaticAssetsDir     string
		PublicStaticDomain  string
		UserRoutesDomain    string
		PortString          string
		SandboxCodePath     string
		SandboxRunnerPath   string
		MigratorScriptPath  string
		AppsPath            string
		AppspacesPath       string
	}
}

// APIVersion is the Dropserver API version that a dropserver app interacts with
type APIVersion int

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

// MetricsI represents the global Metrics interface
type MetricsI interface {
	HostHandleReq(start time.Time)
}

// SandboxManagerI is an interface that describes sm
type SandboxManagerI interface {
	GetForAppSpace(appVersion *AppVersion, appspace *Appspace) chan SandboxI
	StopAppspace(AppspaceID)
}

// SandboxStatus represents the Status of a Sandbox
type SandboxStatus int

const (
	// SandboxStarting sb is starting not ready yet
	SandboxStarting SandboxStatus = iota + 1
	// SandboxReady means it's ready to take incoming requests
	SandboxReady
	// SandboxKilling means the system considers it is going down
	SandboxKilling
	// SandboxDead means it's gone
	SandboxDead
)

// SandboxI describes the interface to a sandbox
type SandboxI interface {
	ID() int
	ExecFn(AppspaceRouteHandler) error
	SendMessage(int, int, []byte) (twine.SentMessageI, error)
	GetTransport() http.RoundTripper
	TiedUp() bool
	LastActive() time.Time
	TaskBegin() chan bool
	Status() SandboxStatus
	SetStatus(SandboxStatus)
	WaitFor(SandboxStatus)
	Start() error
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

// ^^ this should be changed to reflect that User IDs are appspace user ids? (?)
// Maybe this should be AppspaceAuth, because it's quite specific to appspaces and not account admin.

type TimedToken struct {
	Token   string
	Created time.Time
}

//AppspaceLoginToken carries user auth data corresponding to a login token
type AppspaceLoginToken struct {
	AppspaceID    AppspaceID
	AppspaceURL   url.URL
	ProxyID       ProxyID
	LoginToken    TimedToken
	RedirectToken TimedToken
}

// Views interface
type Views interface {
	PrepareTemplates()
	AppspaceLogin(http.ResponseWriter, AppspaceLoginViewData)
	Login(http.ResponseWriter, LoginViewData)
	Signup(http.ResponseWriter, SignupViewData)
	UserHome(http.ResponseWriter)
	Admin(http.ResponseWriter)
}

// AppspaceLoginViewData is used to pass messages and parameters to the login page
type AppspaceLoginViewData struct {
	AppspaceLoginToken string
}

// LoginViewData is used to pass messages and parameters to the login page
type LoginViewData struct {
	Message            string
	Email              string
	AppspaceLoginToken string
}

// SignupViewData is used to pass messages and parameters to the login page
type SignupViewData struct {
	RegistrationOpen bool
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
	Authentication *Authentication
	App            *App
	AppVersion     *AppVersion
	Appspace       *Appspace
	URLTail        string
	RouteConfig    *AppspaceRouteConfig
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
	DBName(string) Error
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

// CookieModel is the interface for storing and retriving cookies
type CookieModel interface {
	PrepareStatements()
	Get(string) (*Cookie, error)
	Create(Cookie) (string, error)
	UpdateExpires(string, time.Time) error
	Delete(string) error
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

	// Maybe we need an IsOwner flag? Or might be able to use UserID for now, since it's there.
	// I think "is owner" comes from appspace users table
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

// SettingsModel is used to get and set settings
type SettingsModel interface {
	Get() (*Settings, Error)
	Set(*Settings) Error
	SetRegistrationOpen(bool) Error
}

// UserModel is the interface for user model
type UserModel interface { //TODO remove domain.Error, also move this
	PrepareStatements()
	Create(string, string) (*User, Error)
	UpdatePassword(UserID, string) Error
	GetFromID(UserID) (*User, Error)
	GetFromEmail(string) (*User, Error)
	GetFromEmailPassword(string, string) (*User, Error)
	GetAll() ([]*User, Error)
	IsAdmin(UserID) bool
	GetAllAdmins() ([]UserID, Error)
	MakeAdmin(UserID) Error
	DeleteAdmin(UserID) Error
}

// UserInvitation represents an invitation for a user to join the DropServer instance
type UserInvitation struct {
	Email string `db:"email" json:"email"`
}

// UserInvitationModel is the interface to the UserInvitation model
type UserInvitationModel interface {
	PrepareStatements()
	GetAll() ([]*UserInvitation, Error)
	Get(email string) (*UserInvitation, Error)
	Create(email string) Error
	Delete(email string) Error
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
	PrevVersion     Version          `json:"prev_version"`
	NextVersion     Version          `json:"next_version"`
	Errors          []string         `json:"errors"`
	VersionMetadata AppFilesMetadata `json:"version_metadata,omitempty"`
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
type AppFilesMetadata struct {
	AppName         string                   `json:"name"`
	AppVersion      Version                  `json:"version"`
	SchemaVersion   int                      `json:"schema"`
	APIVersion      APIVersion               `json:"api_version"`
	Migrations      []int                    `json:"migrations"`
	UserPermissions []AppspaceUserPermission `json:"user_permissions"`
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

// MigrationStatusData reflects the current status of the migrationJob referenced
type MigrationStatusData struct {
	JobID      JobID                `json:"job_id"`
	AppspaceID AppspaceID           `json:"appspace_id"`
	Status     MigrationJobStatus   `json:"status"` // this one is harder, but is it necessary? we have started and finished datetime
	Started    nulltypes.NullTime   `json:"started"`
	Finished   nulltypes.NullTime   `json:"finished"`
	ErrString  nulltypes.NullString `json:"err"`
	CurSchema  int                  `json:"cur_schema"`
}

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

// AppspaceContact is a user of an appspace who is a contact of the owner
type AppspaceContact struct {
	AppspaceID AppspaceID `db:"appspace_id" json:"appspace_id"`
	ContactID  ContactID  `db:"contact_id" json:"contact_id"`
	ProxyID    ProxyID    `db:"proxy_id" json:"proxy_id"`
}

type DbConn interface {
	GetHandle() *sqlx.DB
}

// AppspaceMetaDB manages the files and connections for each appspace's metadata DB
type AppspaceMetaDB interface {
	Create(AppspaceID, int) error
	GetConn(AppspaceID) (DbConn, error)
}

// AppspaceInfoModel holds metadata like current schema and ds api version for the appspace.
type AppspaceInfoModel interface {
	GetSchema() (int, error)
	SetSchema(int) error
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

// V0UserModel serves appsace user data queries at API 0
type V0UserModel interface {
	ReverseServiceI

	// Get a appspace user by proxy id
	// If proxy id does not exist it returns zero-value V0User and nil error
	Get(ProxyID) (V0User, error)
	GetAll() ([]V0User, error)
	Create(proxyID ProxyID, displayName string, permissions []string) error
	Update(proxyID ProxyID, displayName string, permissions []string) error
	Delete(ProxyID) error
}

// V0User is an appspace user as stored in the appspace
type V0User struct {
	ProxyID     ProxyID  `json:"proxy_id"`
	Permissions []string `json:"permissions"`
	DisplayName string   `json:"display_name"`
}

// ReverseServiceI is a common interface for reverse services of all versions
type ReverseServiceI interface {
	HandleMessage(twine.ReceivedMessageI)
}

// TwineService attach to a twine instance to handle two-way communication
// with a remote twine client
type TwineService interface {
	Start(UserID, *twine.Twine)
	HandleMessage(twine.ReceivedMessageI)
}

// Events...

//AppspacePausedEvent is the payload for appspace paused event
type AppspacePausedEvent struct {
	AppspaceID AppspaceID
	Paused     bool
}

// AppspaceRouteEvent carries information about a change in an appspace's routes
type AppspaceRouteEvent struct {
	AppspaceID AppspaceID `json:"appspace_id"`
	Path       string     `json:"path"`
}

//AppspaceStatusEvent indicates readiness of appspace and the reason
type AppspaceStatusEvent struct {
	AppspaceID       AppspaceID `json:"appspace_id"`
	Paused           bool       `json:"paused"`
	TempPaused       bool       `json:"temp_paused"`
	Migrating        bool       `json:"migrating"`
	AppspaceSchema   int        `json:"appspace_schema"`
	AppVersionSchema int        `json:"app_version_schema"`
	Problem          bool       `json:"problem"` // string? To hint at the problem?
}

//AppspaceLogEvent is a log entry
type AppspaceLogEvent struct {
	Time       time.Time  `json:"time"`
	AppspaceID AppspaceID `json:"appspace_id"`
	Source     string     `json:"source"`
	Data       string     `json:"data"` // TODO: or interface? Stirng may be better because it is written in the log as json encoded string.
	Message    string     `json:"message"`
}

// AppspaceRouteHitEvent contains the route that was matched with the request
type AppspaceRouteHitEvent struct {
	Timestamp   time.Time
	AppspaceID  AppspaceID
	Request     *http.Request
	RouteConfig *AppspaceRouteConfig
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
