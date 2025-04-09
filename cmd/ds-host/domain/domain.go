package domain

//go:generate mockgen -destination=mocks.go -package=domain -self_package=github.com/teleclimber/DropServer/cmd/ds-host/domain github.com/teleclimber/DropServer/cmd/ds-host/domain MetricsI,SandboxI,StdInput
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
		TLSPort  uint16 `json:"tls-port"`  // defaults to 443.
		HTTPPort uint16 `json:"http-port"` // defaults to 80.
		NoTLS    bool   `json:"no-tls"`    // do not start HTTPS server
		// TLS cert and key for the HTTPS server (if any).
		// Leave empty if using ManageTLSCertificates
		TLSCert string `json:"tls-cert"`
		TLSKey  string `json:"tls-key"`
	} `json:"server"`
	ExternalAccess struct {
		Scheme    string `json:"scheme"`    // http or https // default to https
		Subdomain string `json:"subdomain"` // for users login // default to dropid
		Domain    string `json:"domain"`
		Port      uint16 `json:"port"` // default to 443
	} `json:"external-access"`
	// TrustCert is used in ds2ds
	TrustCert    string `json:"trust-cert"`
	LocalNetwork struct {
		AllowedIPs []string `json:"allowed-ips"` // Allowed IP addresses, or CIDR ranges.
	} `json:"local-network"`
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
		RuntimeFilesPath string
		SandboxCodePath  string
		AppsPath         string
		AppspacesPath    string
		CertificatesPath string
	}
}

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

// Settings represents some admin-settable parameters
// Some other parameters like tsnet must be fetched separately
type Settings struct {
	RegistrationOpen bool `json:"registration_open" db:"registration_open"`
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
	UserID          UserID `db:"user_id"`
	Email           string
	TSNetIdentifier string `db:"tsnet_identifier"`
	TSNetExtraName  string `db:"tsnet_extra_name"`
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

// AppListingFetch is the app listing along with some fetch metadata
type AppListingFetch struct {
	// FetchDatetime is the time that the fetch was performed
	FetchDatetime time.Time
	// NotModified is true if the remote endpoint returned the Not-Modified header
	NotModified bool
	// NewURL is set if remote endpoint returns a redirect
	// or if listing contains NewURL
	NewURL string
	// Listing is the last successfully fetched app listing
	// Maybe this should not be in here? Get it separately if you actually want the listing data?
	Listing AppListing
	// ListingDatetime for HTTP cache purposes
	ListingDatetime time.Time
	// Etag of fetched listing for caching purposes
	Etag string
	// LatestVersion is the highest stable semver of all versions in listing.
	// I wonder if maybe latest version should be somewhere else?
	// It's not really "fetch"-related. It's interpretation of listing.
	LatestVersion Version
}

// AppURLData contains all metadata related to fetching the app listing
type AppURLData struct {
	AppID AppID `db:"app_id" json:"app_id"`

	// URL of the app listing JSON file
	// It should not need redirecting
	URL string `db:"url" json:"url"`

	// Automatic fetch of the app listing
	Automatic bool `db:"automatic" json:"automatic"`

	// Last fetch attempted
	Last time.Time `db:"last_dt" json:"last_dt"` // Not null, this struct can onle exist after created after inital fetch?
	// LastResult values:
	// - "ok": fetch succeeded with new listing
	// - "not-modified": remote returned that resource not modified
	// - "new-url": remote indicated there is a new url to fetch things from (details?) (But we already have a new url field in DB?)
	// - "error": some error happened. But would love to stash the actual error as well.
	LastResult string `db:"last_result" json:"last_result"`

	// NewURL from which the app listing should be fetched.
	// This is set when the original URL returns a permanent redirect
	// or the "new-url" field is set in the listing
	// and the new URL requires confirmation from the user.
	NewURL string `db:"new_url" json:"new_url"`
	// NewUrlDatetime timestamp of when the new URL was initially discovered (is this necessary?)
	NewUrlDatetime nulltypes.NullTime `db:"new_url_dt" json:"new_url_dt"`

	// ListingDatetime
	ListingDatetime time.Time `db:"listing_dt" json:"listing_dt"`
	// Etag of fetched listing for caching purposes
	Etag string `db:"etag"` // do we need the etag in JSON?

	// LatestVersion is the highest stable semver of all versions in listing.
	LatestVersion Version `db:"latest_version" json:"latest_version"`
}

// AppVersion represents a set of version of an app
// This struct is meant for backend use, like starting a sandbox.
type AppVersion struct {
	AppID       AppID     `db:"app_id" json:"app_id"`
	Version     Version   `db:"version" json:"version"`
	Schema      int       `db:"schema" json:"schema"`
	Entrypoint  string    `db:"entrypoint" json:"entrypoint"`
	Created     time.Time `db:"created" json:"created"`
	LocationKey string    `db:"location_key" json:"-"`
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

type AppGetKey string

// ProcessProblem are consts that enable checking for
// a specific type of problem at processing time.
type ProcessProblem string

const ProblemEmpty ProcessProblem = "empty"
const ProblemInvalid ProcessProblem = "invalid"
const ProblemBig ProcessProblem = "big"
const ProblemSmall ProcessProblem = "small" // maybe roll this into poor experience
const ProblemNotFound ProcessProblem = "not-found"

// ProblemError implies an error took place while processing
const ProblemError ProcessProblem = "error"

// ProblemPoorExperience indicate the value is usable but does
// not meet best practices and affects the user's experience
const ProblemPoorExperience ProcessProblem = "poor-experience"

type ProcessWarning struct {
	Field    string         `json:"field"`     // Field indicates area of problem. It can be the json key from manifest or something else
	Problem  ProcessProblem `json:"problem"`   // Problem for classification
	BadValue string         `json:"bad_value"` // BadValue of field for safe display
	Message  string         `json:"message"`   // Message for user or developer
}

// AppGetMeta has app version data and any errors found in it
type AppGetMeta struct {
	Key         AppGetKey        `json:"key"`
	PrevVersion Version          `json:"prev_version"`
	NextVersion Version          `json:"next_version"`
	Errors      []string         `json:"errors"`
	Warnings    []ProcessWarning `json:"warnings"`
	// VersionManifest is currently the manifest as determined by the app processing steps.
	VersionManifest AppVersionManifest `json:"version_manifest"`
	// AppID of the app if getting a new version, or of the created app if new app
	AppID AppID `json:"app_id"`
}

// AppGetEvent contains updates to an app getter process
type AppGetEvent struct {
	OwnerID UserID    `json:"owner_id"`
	Key     AppGetKey `json:"key"`
	// Done means the entire process is finished, nothing more is going to happen.
	Done bool `json:"done"`
	// Input is non-empty string when user input is needed (like "commit", or "see warnings then continue")
	Input string `json:"input"`
	// Step is user-readable strings that give an indication of the steps taken.
	Step string `json:"step"`
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

// TSNetCommon is the tsnet data that is stored for both
// appspace and user tsnet nodes
type TSNetCommon struct {
	ControlURL string `db:"control_url" json:"control_url"`
	Hostname   string `db:"hostname" json:"hostname"`
	Connect    bool   `db:"connect" json:"connect"`
}

// AppspaceTSNet contains the appspace's tailscale node config data
type AppspaceTSNet struct {
	TSNetCommon
	AppspaceID AppspaceID `db:"appspace_id" json:"-"`
}

type TSNetCreateConfig struct {
	ControlURL string   `json:"control_url"`
	Hostname   string   `json:"hostname"`
	Tags       []string `json:"tags"`
	AuthKey    string   `json:"auth_key"`
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

// AppRoute is route config for appspace as stored with app
type AppRoute struct {
	ID      string            `json:"id"`
	Method  string            `json:"method"`
	Path    AppRoutePath      `json:"path"` // Path is the request path to match
	Auth    AppspaceRouteAuth `json:"auth"`
	Type    string            `json:"type"`    // Type of handler: "function" or "static" for now
	Options AppRouteOptions   `json:"options"` // Options for the route handler
}

// AppRoutePath is the request path twe are seeking to match
type AppRoutePath struct {
	Path string `json:"path"`
	End  bool   `json:"end"` // End of false makes the path a wildcard /path/**
}

// AppRouteOptions is a JSON friendly struct
// that describes the desired handling for the route
type AppRouteOptions struct {
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
type AppspaceUser struct {
	AppspaceID  AppspaceID         `json:"appspace_id"`
	ProxyID     ProxyID            `json:"proxy_id"`
	Auths       []AppspaceUserAuth `json:"auths"`
	DisplayName string             `json:"display_name"`
	Avatar      string             `json:"avatar"`
	Permissions []string           `json:"permissions"`
	Created     time.Time          `json:"created_dt"`
}

type AppspaceUserAuth struct {
	Type       string    `db:"type" json:"type"`
	Identifier string    `db:"identifier" json:"identifier"`
	ExtraName  string    `db:"extra_name" json:"extra_name"`
	Created    time.Time `db:"created" json:"created_dt"`
}

type EditOperation string

const (
	EditOperationNoOp   EditOperation = ""
	EditOperationAdd    EditOperation = "add"
	EditOperationRemove EditOperation = "remove"
)

type EditAppspaceUserAuth struct {
	Type       string        `json:"type"`
	Identifier string        `json:"identifier"`
	ExtraName  string        `json:"extra_name"`
	Operation  EditOperation `json:"operation"`
}

// TSNetStatus is info about the tsnet server
type TSNetStatus struct {
	ControlURL      string                  `json:"control_url"`
	URL             string                  `json:"url,omitempty"`
	IP4             string                  `json:"ip4,omitempty"`
	IP6             string                  `json:"ip6,omitempty"`
	ListeningTLS    bool                    `json:"listening_tls,omitempty"` // If the TLS server is on for this node.
	Tailnet         string                  `json:"tailnet,omitempty"`
	KeyExpiry       *time.Time              `json:"key_expiry,omitempty"`
	Name            string                  `json:"name,omitempty"` // DNS name, which is sometimes machine name
	HTTPSAvailable  bool                    `json:"https_available"`
	MagicDNSEnabled bool                    `json:"magic_dns_enabled"`
	Tags            []string                `json:"tags"`
	ErrMessage      string                  `json:"err_message,omitempty"`
	State           string                  `json:"state,omitempty"` // State from tsnet. But not ideal. Use "connected" instead of "running"
	Usable          bool                    `json:"usable"`
	BrowseToURL     string                  `json:"browse_to_url,omitempty"`
	LoginFinished   bool                    `json:"login_finished,omitempty"`
	Warnings        map[string]TSNetWarning `json:"warnings,omitempty"`
	Transitory      string                  `json:"transitory"` // "" "connecting" or "disconnecting"
}
type TSNetWarning struct {
	Title               string `json:"title"`
	Text                string `json:"text"`
	Severity            string `json:"severity"`
	ImpactsConnectivity bool   `json:"impacts_connectivity"`
}
type TSNetAppspaceStatus struct {
	TSNetStatus
	AppspaceID AppspaceID `json:"appspace_id"`
}

type TSNetUserDevice struct {
	ID          string     `json:"id"`                  // Node stable id?
	Name        string     `json:"name"`                // Node.Name() That's DNS name of device, though empty for sharee
	Online      *bool      `json:"online,omitempty"`    // if nil then it's unknown or not knowable
	LastSeen    *time.Time `json:"last_seen,omitempty"` // nil if it's never been online or no permission to know. if online is true, ignore.
	OS          string     `json:"os"`
	DeviceModel string     `json:"device_model"`
	App         string     `json:"app"` // to disambibuate ts client from tsnet or something. Interesting?
}

// TSNetPeerUser provides details of a user on a tailnet
// Essentially equivalent to tailcfg.UserProfile, but includes backend url
type TSNetPeerUser struct {
	ID          string            `json:"id"` // tsnet user id
	LoginName   string            `json:"login_name"`
	DisplayName string            `json:"display_name"`
	Sharee      bool              `json:"sharee"`
	Devices     []TSNetUserDevice `json:"devices"`
	ControlURL  string            `json:"control_url"`
	FullID      string            `json:"full_id"`
	//ProfilePicURL string //this isn't right? Create a separate route for fetching avatars for these users.
}

// V0RouteModel serves route data queries at version 0
// type V0RouteModel interface {
// 	ReverseServiceI

// 	Create(methods []string, url string, auth AppspaceRouteAuth, handler AppspaceRouteHandler) error

// 	// Get returns all routes that
// 	// - match one of the methods passed, and
// 	// - matches the routePath exactly (no interpolation is done to match sub-paths)
// 	Get(methods []string, routePath string) (*[]AppspaceRouteConfig, error)
// 	GetAll() (*[]AppspaceRouteConfig, error)
// 	GetPath(string) (*[]AppspaceRouteConfig, error)

// 	Delete(methods []string, url string) error

// 	// Match finds the route that should handle the request
// 	// The path will be broken into parts to find the subset path that matches.
// 	// It returns (nil, nil) if no matches found
// 	Match(method string, url string) (*AppspaceRouteConfig, error)
// }

// // AppspaceRouteModels returns models of the desired version
// type AppspaceRouteModels interface {
// 	GetV0(AppspaceID) V0RouteModel
// }

// ReverseServiceI is a common interface for sandbox services
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
	OwnerID          UserID     `json:"owner_id"`
	AppspaceID       AppspaceID `json:"appspace_id"`
	Paused           bool       `json:"paused"`
	TempPaused       bool       `json:"temp_paused"`
	TempPauseReason  string     `json:"temp_pause_reason"`
	AppspaceSchema   int        `json:"appspace_schema"`
	AppVersionSchema int        `json:"app_version_schema"`
	Problem          bool       `json:"problem"` // string? To hint at the problem?
}

type AppspaceTSNetModelEvent struct {
	Deleted bool `json:"deleted"`
	AppspaceTSNet
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
	Timestamp   time.Time
	AppspaceID  AppspaceID
	Request     *http.Request
	RouteConfig *AppRoute // this needs to be normalized. use generic app route, not versioned
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
