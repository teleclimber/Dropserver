package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/record"
	"tailscale.com/client/tailscale"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/health"
	"tailscale.com/ipn"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

//

// AppspaceTSNode refs an appspace's tsnet node
type AppspaceTSNode struct {
	Config                *domain.RuntimeConfig
	AppspaceLocation2Path interface {
		TailscaleNodeStore(locationKey string) string
	}
	AppspaceModel interface {
		GetFromID(domain.AppspaceID) (*domain.Appspace, error)
	}
	AppspaceRouter            http.Handler
	AppspaceTSNetStatusEvents interface {
		Send(data domain.TSNetAppspaceStatus)
	}

	appspaceID domain.AppspaceID
	// ^^ actually we need to full appspace? Need loc key, domain / ts net name, ...
	// but ts net name is only the desired name, which is used once if creating the node. after that it's data from teh outside
	ownerID domain.UserID
	// ^^ also need owner id to send with notification

	tsnetServer *tsnet.Server
	ln80        net.Listener
	ln443       net.Listener

	busWatcherCtxCancel context.CancelFunc

	// status
	// lock?
	nodeStatus tsNodeStatus

	usersMux sync.Mutex
	users    []tsUser
}

// Is this create node or is this start node?
func (n *AppspaceTSNode) createNode(domainName string) error {
	logger := n.getLogger("createNode").AppspaceID(n.appspaceID)

	appspace, err := n.AppspaceModel.GetFromID(n.appspaceID)
	if err != nil {
		return err
	}

	name := strings.Split(domainName, ".")[0]

	s := new(tsnet.Server)
	n.tsnetServer = s

	// To use headscale or an other alternative backend set the control URL:
	// s.ControlURL = "...m"
	// s.AuthKey = "..."

	s.Dir = n.AppspaceLocation2Path.TailscaleNodeStore(appspace.LocationKey) // TODO prefix this with control plane domain
	s.Hostname = name                                                        // Set the name? or is taht only used when we are creating the node?

	s.Logf = nil
	s.UserLogf = func(format string, args ...any) {
		if !strings.Contains(format, "restart with TS_AUTHKEY set") {
			logger.Clone().AddNote("tsnet UserLogf").Log(fmt.Sprintf(format, args...))
		}
	}

	lc, err := s.LocalClient()
	if err != nil {
		logger.Clone().AddNote("LocalClient()").Error(err)
		return err
	}

	bwCtx, bwCancel := context.WithCancel(context.Background())
	n.busWatcherCtxCancel = bwCancel
	busWatcher, err := lc.WatchIPNBus(bwCtx, 0)
	if err != nil {
		logger.Clone().AddNote("WatchIPNBus()").Error(err)
		return err
	}

	go func() { // this should be a separate func
		for {
			newData, err := busWatcher.Next()
			if err != nil {
				logger.Clone().AddNote("busWatcher.Next()").Error(err)
				break
			}
			magicDNS := n.nodeStatus.magicDNS
			if newData.NetMap != nil {
				status, err := lc.Status(context.Background())
				if err != nil {
					logger.Clone().AddNote("buswatcher lc.Status()").Error(err)
				}
				if status != nil && status.CurrentTailnet != nil {
					magicDNS = status.CurrentTailnet.MagicDNSEnabled
				}
				// note that netmap contains much more than peers!
				// it also contains UserProfiles:
				fmt.Println("user profiles")
				for id, p := range newData.NetMap.UserProfiles {
					fmt.Println(id, p)
				}
				n.ingestPeers(lc, newData.NetMap.Peers)
				// send notification in goroutine
			}
			if n.nodeStatus.ingest(newData, magicDNS) {
				n.sendStatus()
			}

			n.startStopHTTPS()
		}
		err = busWatcher.Close()
		if err != nil {
			logger.Clone().AddNote("busWatcher.Close error").Error(err)
		}
	}()

	n.ln80, err = s.Listen("tcp", ":80")
	if err != nil {
		logger.Clone().AddNote("Listen()").Error(err)
		return err
	}
	go n.handler(n.ln80)

	n.sendStatus()

	n.startStopHTTPS()

	return nil
}

func (n *AppspaceTSNode) startStopHTTPS() {
	if n.ln443 == nil && n.nodeStatus.magicDNS && n.nodeStatus.httpsAvailable {
		fmt.Println(n.appspaceID, "starting HTTPS server")
		var err error
		n.ln443, err = n.tsnetServer.ListenTLS("tcp", ":443")
		if err != nil {
			n.getLogger("startStopHTTPS ListenTLS()").Error(err)
			return
		}
		go n.handler(n.ln443)
		n.sendStatus()
	} else if n.ln443 != nil && (!n.nodeStatus.magicDNS || !n.nodeStatus.httpsAvailable) {
		fmt.Println(n.appspaceID, "stopping HTTPS server")
		err := n.ln443.Close()
		if err != nil {
			n.getLogger("startStopHTTPS ln443.Close()").Error(err)
		}
		n.ln443 = nil
		n.sendStatus()
	}
}

func (n *AppspaceTSNode) handler(ln net.Listener) {
	err := http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggerFn := n.getLogger("http.Serve").AppspaceID(n.appspaceID).Clone
		loggerFn().Debug("tsnet got request")

		// Set the appspace in the context for use in appspace router. We're handling
		// requests over time, so reload the appspace every time in case it changes.
		appspace, err := n.AppspaceModel.GetFromID(n.appspaceID)
		if err != nil {
			loggerFn().AddNote("AppspaceModel.GetFromID").Error(err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if appspace == nil {
			loggerFn().AddNote("AppspaceModel.GetFromID").Log("no appspace returned")
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		r = r.WithContext(domain.CtxWithAppspaceData(r.Context(), *appspace))

		lc, err := n.tsnetServer.LocalClient()
		if err != nil {
			loggerFn().AddNote("tsnetServer.LocalClient").Error(err)
			http.Error(w, "tsnet error", http.StatusInternalServerError)
			return
		}
		who, err := lc.WhoIs(r.Context(), r.RemoteAddr)
		if err != nil {
			loggerFn().AddNote("lc.WhoIs").Error(err)
			http.Error(w, "tsnet whois error", http.StatusInternalServerError)
			return
		}
		// Set the user id from [tail|head]scale for use in determining the ProxyID later
		r = r.WithContext(domain.CtxWithTSNetUserID(r.Context(), who.UserProfile.ID.String()))

		n.AppspaceRouter.ServeHTTP(w, r)
	}))
	if err != nil {
		n.getLogger("handler").AddNote("http.Serve()").Error(err)
		return
	}
}

func (n *AppspaceTSNode) stop() {
	if n.tsnetServer == nil {
		return
	}
	err := n.tsnetServer.Close()
	if err != nil {
		n.getLogger("stop Close()").Error(err)
	}
	if n.busWatcherCtxCancel != nil {
		n.busWatcherCtxCancel()
	}
}

// might need a context here that gets passed to WhoIsNodeKey
// and maybe check before each iteration of the loop.
func (n *AppspaceTSNode) ingestPeers(lc *tailscale.LocalClient, peers []tailcfg.NodeView) {
	n.usersMux.Lock()
	defer n.usersMux.Unlock()
	n.users = make([]tsUser, 0)

	loggerFn := n.getLogger("ingestPeers").AppspaceID(n.appspaceID).Clone
	for _, nv := range peers {
		var userID string
		who, err := lc.WhoIsNodeKey(context.Background(), nv.Key())
		if err != nil {
			err = fmt.Errorf("whoIsNodeKey error: %w", err)
			loggerFn().Error(err)
			// maybe set as lastUsersError
			continue
		} else if who.UserProfile != nil {
			userID = who.UserProfile.ID.String()
			if who.UserProfile.ID.String() != nv.User().String() {
				// that's something that I'd like to know about!
				err = fmt.Errorf("who.UserProfile.ID.String() != nv.User().String(): %s != %s", who.UserProfile.ID.String(), nv.User().String())
				loggerFn().Error(err)
				// add error to stack?
			}
		}

		if userID != "" {
			found := false
			u := tsUser{}
			for i, u := range n.users {
				if u.id == userID {
					found = true
					n.users[i].ingest(nv, who)
					break
				}
			}
			if !found {
				u = tsUser{}
				u.ingest(nv, who)
				n.users = append(n.users, u)
			}
		}
	}
	for i, u := range n.users {
		fmt.Println("user", i, u.id, u.displayName, u.loginName, u.sharee, u.nodes)
	}
}

func (n *AppspaceTSNode) sendStatus() {
	n.AppspaceTSNetStatusEvents.Send(n.getStatus())
}

func (n *AppspaceTSNode) getStatus() domain.TSNetAppspaceStatus {
	stat := n.nodeStatus.asDomain()
	stat.AppspaceID = n.appspaceID
	stat.OwnerID = n.ownerID
	stat.ListeningTLS = n.ln443 != nil
	stat.URL = n.buildURL()
	return stat
}

func (n *AppspaceTSNode) buildURL() string {
	proto := "http"
	if n.ln443 != nil {
		proto = "https"
	}
	addr := n.nodeStatus.ip4
	if n.nodeStatus.magicDNS {
		addr = strings.TrimRight(n.nodeStatus.dnsName, ".")
	}
	return fmt.Sprintf("%s://%s", proto, addr)
}

func (n *AppspaceTSNode) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("AppspaceTSNode").AppspaceID(n.appspaceID)
	if note != "" {
		r.AddNote(note)
	}
	return r
}

type tsNodeStatus struct {
	dnsName        string
	tailnet        string
	magicDNS       bool
	httpsAvailable bool
	ip4            string
	ip6            string
	errMessage     string
	state          string
	browseToURL    string
	loginFinished  bool
	warnings       map[string]health.UnhealthyState
	// TODO add node expiry
}

// ingest note that any part of the status is non-nil only if it changed.
func (n *tsNodeStatus) ingest(data ipn.Notify, magicDNS bool) (changed bool) {
	if data.NetMap != nil {
		changed = true                 // but maybe not?
		n.dnsName = data.NetMap.Name   // Name is "dns name with trailing dot"
		n.tailnet = data.NetMap.Domain // Domain is "tailnet name"
	}
	if magicDNS != n.magicDNS {
		n.magicDNS = magicDNS
		changed = true
	}
	if data.NetMap != nil {
		https := len(data.NetMap.DNS.CertDomains) != 0
		if https != n.httpsAvailable {
			n.httpsAvailable = https
			changed = true
		}
		for _, a := range data.NetMap.SelfNode.Addresses().All() {
			if a.IsSingleIP() && a.IsValid() {
				if a.Addr().Is4() {
					ip4 := a.Addr().String()
					if n.ip4 != ip4 {
						n.ip4 = ip4
						changed = true
					}
				}
				if a.Addr().Is6() {
					ip6 := a.Addr().String()
					if n.ip6 != ip6 {
						n.ip6 = ip6
						changed = true
					}
				}
			}
		}
	}
	if data.ErrMessage != nil {
		changed = true
		n.errMessage = *data.ErrMessage
	}
	if data.State != nil {
		changed = true
		n.state = data.State.String()
	}
	if data.BrowseToURL != nil {
		changed = true
		n.browseToURL = *data.BrowseToURL
	}
	if data.LoginFinished != nil {
		changed = true
		n.loginFinished = true
	}
	n.warnings = make(map[string]health.UnhealthyState)
	if data.Health != nil && len(data.Health.Warnings) != 0 {
		changed = true
		for w, us := range data.Health.Warnings {
			n.warnings[string(w)] = us
			// warnings is a whole thing
			// Including TimeToVisible! and Depends on which change which warnings are displayed
		}
	}
	return
}

func (n *tsNodeStatus) asDomain() domain.TSNetAppspaceStatus {
	warnings := make(map[string]domain.TSNetWarning)
	for w, d := range n.warnings {
		warnings[w] = domain.TSNetWarning{
			Title:               d.Title,
			Text:                d.Text,
			Severity:            string(d.Severity),
			ImpactsConnectivity: d.ImpactsConnectivity,
		}
	}
	ret := domain.TSNetAppspaceStatus{
		Tailnet:         n.tailnet,
		MagicDNSEnabled: n.magicDNS,
		Name:            n.dnsName,
		IP4:             n.ip4,
		IP6:             n.ip6,
		HTTPSAvailable:  n.httpsAvailable,
		ErrMessage:      n.errMessage,
		State:           n.state,
		BrowseToURL:     n.browseToURL,
		LoginFinished:   n.loginFinished,
		Warnings:        warnings,
	}
	return ret
}

type tsUser struct {
	id          string
	displayName string
	loginName   string
	picURL      string
	sharee      bool
	nodes       []tsUserNode
}

func (u *tsUser) ingest(nv tailcfg.NodeView, who *apitype.WhoIsResponse) {
	u.id = who.UserProfile.ID.String()
	u.displayName = who.UserProfile.DisplayName
	u.loginName = who.UserProfile.LoginName
	u.picURL = who.UserProfile.ProfilePicURL
	u.sharee = nv.Hostinfo().ShareeNode()

	if u.nodes == nil {
		u.nodes = make([]tsUserNode, 0)
	}
	u.nodes = append(u.nodes, ingestNode(nv))
}

func ingestNode(nv tailcfg.NodeView) tsUserNode {
	return tsUserNode{
		id:       string(nv.StableID()),
		name:     nv.Name(),
		online:   nv.Online(),
		lastSeen: nv.LastSeen(),
		os:       nv.Hostinfo().OS(), // lots more to do here
		app:      nv.Hostinfo().App(),
	}
}

type tsUserNode struct {
	id   string // Node stable id?
	name string // Node.Name() That's DNS name of device, though empty for sharee
	//computedName string    // or computedNameWithHost
	online   *bool      // if nil then it's unknown or not knowable
	lastSeen *time.Time // nil if it's never been online or no permission to know. if online is true, ignore.
	os       string     // hostinfo.OS , and OsVersion
	app      string     // to disambibuate ts client from tsnet or something. Interesting?
	//location     string    // maybe?
}
