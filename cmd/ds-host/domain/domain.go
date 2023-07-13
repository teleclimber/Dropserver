package domain

//go:generate mockgen -destination=mocks.go -package=domain -self_package=github.com/teleclimber/DropServer/cmd/ds-host/domain github.com/teleclimber/DropServer/cmd/ds-host/domain MetricsI,SandboxI,V0RouteModel,AppspaceRouteModels,StdInput
// ^^ remember to add new interfaces to list of interfaces to mock ^^

import (
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/teleclimber/DropServer/internal/nulltypes"
	"github.com/teleclimber/twine-go/twine"
)

// Reverse Proxy:
// - port: port to connect to for getting lists of domains and reloading certs
// (later can add specifically HAP API etc...)

// Domains:
// There is the host of the admin / user site (dropid.dropserver.develop)
// Then there is the domain where appspaces are created, which may or may not be the same.
// main-domain (for user and admin stuff):
// - hostname
// - generate cert? (unless no-tls) (certificate-management must be on)
// -> ds-host at startup needs to read about all domains on system and check that none is the child of another if auth is involved.
//    .. and then it needs to notify proxy of all domains and then start generating any certificates

// RuntimeConfig represents the variables that can be set at runtime
// Or at least set via config file or cli flags that get read once
// upon starting ds-host.
// This is for server-side use only.
type RuntimeConfig struct {
	DataDir string `json:"data-dir"`
	Server  struct {
		TLSPort  int16 `json:"tls-port"`  // defaults to 443.
		HTTPPort int16 `json:"http-port"` // defaults to 80.
		NoTLS    bool  `json:"no-tls"`    // do not start HTTPS server
		// TLS cert and key for the HTTPS server (if any).
		// Leave empty if using ManageTLSCertificates
		TLSCert string `json:"tls-cert"`
		TLSKey  string `json:"tls-key"`
	} `json:"server"`
	ExternalAccess struct {
		Scheme    string `json:"scheme"`    // http or https // default to https
		Subdomain string `json:"subdomain"` // for users login // default to dropid
		Domain    string `json:"domain"`
		Port      int16  `json:"port"` // default to 443
	} `json:"external-access"`
	// TrustCert is used in ds2ds
	TrustCert             string `json:"trust-cert"`
	ManageTLSCertificates struct {
		Enable              bool   `json:"enable"`
		Email               string `json:"acme-account-email"`
		IssuerEndpoint      string `json:"issuer-endpoint"`       // default use lets encrypt?
		RootCACertificate   string `json:"root-ca-certificate"`   // only needed if ds-host does TLS termination, right? Also apparently only used with issuer endpoint
		DisableOCSPStapling bool   `json:"disable-ocsp-stapling"` // default false
	} `json:"manage-certificates"`
	Sandbox struct {
		SocketsDir    string   `json:"sockets-dir"` // do we really need this? could we not put it in DataDir/sockets?
		UseBubblewrap bool     `json:"use-bubblewrap"`
		BwrapMapPaths []string `json:"bwrap-map-paths"` // for bwrap to be able to run Deno
		UseCGroups    bool     `json:"use-cgroups"`
		CGroupMount   string   `json:"cgroup-mount"`
		// MemoryBytesMb is the memory.high value for the cgroup that is parent of all sandboxe cgroups
		MemoryHighMb int `json:"memory-high-mb"`
		Num          int `json:"num"`
	} `json:"sandbox"`
	Log        string `json:"log"`
	Prometheus struct {
		Enable bool   `json:"enable"`
		Port   uint16 `json:"port"`
	} `json:"prometheus"`
	// Exec contains values determined at runtime
	// These are not settable via json.
	Exec struct {
		CmdVersion       string
		PortString       string
		UserRoutesDomain string
		DenoFullPath     string
		DenoVersion      string
		SandboxCodePath  string
		AppsPath         string
		AppspacesPath    string
		CertificatesPath string
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

// CGroupLimits specifies the limits to appply to sandboxes via cgroup controllers
type CGroupLimits struct {
	MemoryHigh int
}

type CGroupData struct {
	CpuUsec     int
	MemoryBytes int
	IOBytes     int
	IOs         int
}

// SandboxRunData contains the metrics of a sandbox run
type SandboxRunData struct {
	TiedUpMs      int `db:"tied_up_ms" json:"tied_up_ms"`
	CpuUsec       int `db:"cpu_usec" json:"cpu_usec"`
	MemoryByteSec int `db:"memory_byte_sec" json:"memory_byte_sec"`
	IOBytes       int `db:"io_bytes" json:"io_bytes"`
	IOs           int `db:"io_ops" json:"io_ops"`
}

type SandboxRun struct {
	SandboxRunIDs
	SandboxRunData
	Start time.Time          `db:"start" json:"start"`
	End   nulltypes.NullTime `db:"end" json:"end"`
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
// ...which is all fine, but it means cookies have to be tweaked as well
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

// V0AppspaceLoginToken carries user auth data corresponding to a login token
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
	HasSetupKey      bool
	FormAction       string
	Message          string
	Email            string
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
	OwnerID UserID    `db:"owner_id" json:"owner_id"`
	AppID   AppID     `db:"app_id" json:"app_id"`
	Created time.Time `db:"created" json:"created_dt"`
}

// AppVersion represents a set of version of an app
// This struct is meant for backend use, like starting a sandbox.
type AppVersion struct {
	AppID       AppID      `db:"app_id" json:"app_id"`
	Version     Version    `db:"version" json:"version"`
	APIVersion  APIVersion `db:"api" json:"-"`
	Schema      int        `db:"schema" json:"schema"`
	Entrypoint  string     `db:"entrypoint" json:"entrypoint"`
	Created     time.Time  `db:"created" json:"created"`
	LocationKey string     `db:"location_key" json:"-"`
	// consider adding:
	// - migrations (summarized, like up-from, down-to?, eventually requried for properly running migrations)
	// -> maybe migrations could be a separate query, or load the whole manifest when that comes up.
}

type AppVersionUI struct {
	AppID            AppID            `db:"app_id" json:"app_id"`
	Name             string           `db:"name" json:"name"`
	Version          Version          `db:"version" json:"version"`
	Schema           int              `db:"schema" json:"schema"`
	Created          time.Time        `db:"created" json:"created_dt"`
	ShortDescription string           `db:"short_desc" json:"short_desc"`
	AccentColor      string           `db:"color" json:"color"`
	Authors          []ManifestAuthor `json:"authors"`        //maybe truncate to three authors and send number of authors too?
	Code             string           `db:"code" json:"code"` // code repo
	Website          string           `db:"website" json:"website"`
	Funding          string           `db:"funding" json:"funding"`
	ReleaseDate      string           `db:"release_date" json:"release_date"`
	License          string           `db:"license" json:"license"`
}

// MetadataLinkedFile for when metadata references a file. used by:
// - Icon (must be included?),
// - Long Description (should be included),
// - License File (either package with app or let the spdx speak for itself.)
// - Release Notes. Does not fit, except that the notes for the current release make sense to include.

type MetadataLinkedFile struct {
	Source string // The originally specified location
	Local  string // The location based off of app metadata dir of some sort? Empty if not loaded
	// Should we have a retrieveal date?
}

type ManifestAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

type AppVersionManifest struct {
	// Name of the application. Optional.
	Name string `json:"name"` // L10N? Also should have omitempty?
	// ShortDescription is a 10-15 words used to tell prsopective users what this is does.
	ShortDescription string `json:"short-description"` // I18N string.
	// Version in semver format. Required.
	Version Version `json:"version"`
	// Entrypoint is the script that runs the app. Optional. If ommitted system will look for app.ts or app.js.
	Entrypoint string `json:"entrypoint"`
	// Schema is the verion of the appspace data schema.
	// This is determined automatically by the system.
	Schema int `json:"schema"`
	// Migrations is list of migrations provided by this app version
	Migrations []MigrationStep `json:"migrations"`

	// Icon is a package-relative path to an icon file to display within the installer instance UI.
	Icon string `json:"icon"`
	//AccentColor is a CSS color used to differentiate the app in the Dropserver UI
	AccentColor string `json:"accent-color"`

	// Both of these are not currently handled.
	// Description  string `json:"description"`   // link to markdown file? I18N??
	// ReleaseNotes string `json:"release-notes"` // link to release notes markdown?

	// Authors
	Authors []ManifestAuthor `json:"authors"`

	// Code is the URL of the code repository
	Code string `json:"code"`
	// Website for the app
	Website string `json:"website"`
	// Funding website or site where funding situation is explained
	Funding string `json:"funding"` // should maybe not be a string only...
	// License in SPDX string form
	License string `json:"license"`
	// LicenseFile is a package-relative path to a txt file containing the license text.
	LicenseFile string `json:"license-file"` // Rel path to license file within package.

	//ReleaseDate YYYY-MM-DD of software release date. Should be set automatically by packaging code.
	ReleaseDate string `json:"release-date"` // date of packaging.

	// Size of the installed package in bytes (except that additional space will be taken up when fetching remote modules if applicable)
	// Although maybe the actual installed size can be measured by the packaging system?
	// Size int `json:"size"`
}

type AppGetKey string

// AppGetMeta has app version data and any errors found in it
type AppGetMeta struct {
	Key             AppGetKey          `json:"key"`
	PrevVersion     Version            `json:"prev_version"`
	NextVersion     Version            `json:"next_version"`
	Errors          []string           `json:"errors"`
	Warnings        map[string]string  `json:"warnings"`
	VersionManifest AppVersionManifest `json:"version_manifest,omitempty"`
}

// AppGetEvent contains updates to an app getter process
type AppGetEvent struct {
	Key   AppGetKey `json:"key"`
	Done  bool      `json:"done"`
	Error bool      `json:"error"` // TODO maybe add Warning flag so that event recipeints can act accordingly? Or remove Error because every caller should just get the full dump of the process?
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
	// TODO Add more stuff, particularly about app identifier and version
	// Or not!
}

// AppspaceUserPermission describes a permission that can be granted to
// a user, or via other means. The name and description are user-facing,
// the key is used internally.
type AppspaceUserPermission struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

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

// AppspacePausedEvent is the payload for appspace paused event
type AppspacePausedEvent struct {
	AppspaceID AppspaceID
	Paused     bool
}

// AppspaceRouteEvent carries information about a change in an appspace's routes
// type AppspaceRouteEvent struct {
// 	AppspaceID AppspaceID `json:"appspace_id"`
// 	Path       string     `json:"path"`
// }

// AppspaceStatusEvent indicates readiness of appspace and the reason
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
