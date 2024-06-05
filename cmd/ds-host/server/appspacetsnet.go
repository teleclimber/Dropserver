package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"tailscale.com/client/tailscale"
	"tailscale.com/tsnet"
)

type AppspaceTSNet struct {
	Config         *domain.RuntimeConfig `checkinject:"required"`
	AppspaceRouter http.Handler          `checkinject:"required"` // maybe?
	UserModel      interface {
		GetAll() ([]domain.User, error)
	} `checkinject:"required"`
	AppspaceModel interface {
		// Need to start ts net for all appspaces...
		// Except it depends on whether ts is enabled for instance, user, appspace
		// also appspace may be paused
		// There is no get all appspaces
		// Maybe for now we select all users, then get their appspaces
		GetForOwner(userID domain.UserID) ([]*domain.Appspace, error)
	} `checkinject:"required"`
	AppspaceLocation2Path interface {
		TailscaleNodeStore(locationKey string) string
	} `checkinject:"required"`

	serversMux sync.Mutex
	servers    map[domain.AppspaceID]*tsnet.Server
}

func (a *AppspaceTSNet) StopAll() {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	for _, s := range a.servers {
		go s.Close()
	}
}

func (a *AppspaceTSNet) Init() {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	a.servers = make(map[domain.AppspaceID]*tsnet.Server)
}

func (a *AppspaceTSNet) StartAll() error {
	users, err := a.UserModel.GetAll()
	if err != nil {
		return err
	}
	for _, u := range users {
		appspaces, err := a.AppspaceModel.GetForOwner(u.UserID)
		if err != nil {
			return err
		}
		for _, as := range appspaces {
			a.createNode(*as)
		}
	}
	return nil
}

// Is this create node or is this start node?
func (a *AppspaceTSNet) createNode(appspace domain.Appspace) error { // what do we pass in?
	logger := a.getLogger("createNode").AppspaceID(appspace.AppspaceID)

	name := strings.Split(appspace.DomainName, ".")[0]

	s := new(tsnet.Server)

	a.addServer(appspace.AppspaceID, s)

	// Use headscale by specifiying the URL
	// Although note that headscale can only work for http.
	// Comes from config
	// s.ControlURL = usrl from config...

	s.Dir = a.AppspaceLocation2Path.TailscaleNodeStore(appspace.LocationKey)
	// if this is create then create the dir?
	//log.Printf("storage dir: %s", s.Dir)
	s.Hostname = name // Set the name? or is taht only used when we are creating the node?
	s.Logf = log.New(os.Stderr, fmt.Sprintf("[tsnet:%s] ", name), log.LstdFlags).Printf

	lc, err := s.LocalClient()
	if err != nil {
		logger.AddNote("LocalClient()").Error(err)
		return err
	}
	// printStatus(name, lc)
	// go pollStatus(name, lc)	// todo polling will be required to keep up with the status and get the login URL

	ln, err := s.Listen("tcp", ":80")
	//ln, err := s.ListenTLS("tcp", ":443")
	//ln, err := s.ListenFunnel("tcp", ":443")
	if err != nil {
		// We have gotten an error here for funnel: Funnel not available; "funnel" node attribute not set.
		logger.AddNote("Listen[TLS|Funnel]()").Error(err)
		return err
	}

	logger.Debug("Starting tsnet server")

	go func() {
		err = http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Debug("tsnet got request")

			// Set the appspace in the context for use in appspace router.
			r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), appspace))

			who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			// Set the user id from [tail|head]scale for use in determining the ProxyID later
			r = r.WithContext(domain.CtxWithTsnetUserID(r.Context(), who.UserProfile.ID.String()))

			a.AppspaceRouter.ServeHTTP(w, r)

			// fmt.Fprintf(w, "<html><body><h1>Hello from %s!</h1>\n", name)
			// fmt.Fprintf(w, "<p>You are <b>%s</b> aka %s, id=%s from <b>%s</b> (%s)</p>",
			// 	html.EscapeString(who.UserProfile.LoginName),
			// 	html.EscapeString(who.UserProfile.DisplayName),
			// 	html.EscapeString(who.UserProfile.ID.String()),
			// 	html.EscapeString(firstLabel(who.Node.ComputedName)),
			// 	r.RemoteAddr)
			// fmt.Fprintf(w, "<img src='%s' />", who.UserProfile.ProfilePicURL)
		}))
		if err != nil {
			logger.AddNote("http.Serve()").Error(err)
			return
		}
	}()

	return nil
}

func (a *AppspaceTSNet) addServer(appspaceID domain.AppspaceID, s *tsnet.Server) {
	// probably need to panic or somehow handle if a server is already present for that appspace ID.
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	if _, exists := a.servers[appspaceID]; exists {
		panic("server already exists for appspace id. Handle this better") // TODO
	}
	a.servers[appspaceID] = s
}

func (a *AppspaceTSNet) rmServer(appspaceID domain.AppspaceID) {
	a.serversMux.Lock()
	defer a.serversMux.Unlock()
	delete(a.servers, appspaceID)
}

func firstLabel(s string) string {
	s, _, _ = strings.Cut(s, ".")
	return s
}

func printStatus(name string, lc *tailscale.LocalClient) {
	status, err := lc.Status(context.Background())
	if err != nil {
		reportError(name, "lc.Status", err)
	}
	fmt.Println("SELF HostName", status.Self.HostName)
	fmt.Println("SELF DNSName", status.Self.DNSName)
	fmt.Println("Self:", status.Self)
	fmt.Println("STATUS", name, status.BackendState, status.AuthURL, "cert domains:", status.CertDomains, status.User)
	for n, ps := range status.Peer {
		fmt.Println(n, ps.HostName, ps.DNSName, ps.OS, ps.UserID, "/", ps.AltSharerUserID, ps.ShareeNode)
	}
	fmt.Println("CAPS", status.Self.CapMap)
}

func pollStatus(name string, lc *tailscale.LocalClient) {
	c := time.Tick(5 * time.Second)
	for range c {
		printStatus(name, lc)
	}
}

func reportError(name string, op string, err error) {
	fmt.Println("ERROR ", name, op, err.Error())
}

func (m *AppspaceTSNet) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceTSNet")
	if note != "" {
		r.AddNote(note)
	}
	return r
}
