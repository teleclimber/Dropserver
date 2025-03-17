package server

import (
	"context"
	"errors"
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
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/tsnet"
)

type tsNodeConfig struct {
	controlURL string
	hostname   string
	connect    bool
	authKey    string
	tags       []string
}

// TSNetNode ref controls a tsnet node for use in http serving
type TSNetNode struct {
	Config            *domain.RuntimeConfig
	Router            http.Handler
	TSNetStatusEvents interface {
		Send(data domain.TSNetStatus)
	}
	TSNetPeersEvents interface {
		Send()
	}

	hasAppspaceID bool
	appspaceID    domain.AppspaceID // used for logging and http handler

	tsnetDir string

	desiredConfig tsNodeConfig
	deleteNode    bool

	servMux     sync.Mutex
	tsnetServer *tsnet.Server
	ln80        net.Listener
	ln443       net.Listener

	busWatcherCtxCancel context.CancelFunc

	nodeStatus tsNodeStatus

	usersMux sync.Mutex
	users    []tsUser
}

func (n *TSNetNode) setConfig(config tsNodeConfig) {
	if n.deleteNode {
		return
	}
	n.servMux.Lock()
	defer n.servMux.Unlock()
	n.desiredConfig = config
	if n.tsnetServer == nil { // nil implies fully stopped and ready to start again with new config
		if config.connect {
			go n.startNode()
		}
	} else {
		if !config.connect {
			n.getLogger("setConfig").Debug("connect:false, calling stop")
			go n.stop()
		} else {
			if config.controlURL != n.tsnetServer.ControlURL {
				// restart. But this requires much more because the locally saved files must be deleted.
				// This is likely better handled externally, so just stop the node if ith's the wrong controlURL.
				n.getLogger("setConfig").Debug(fmt.Sprintf("control url changed %s -> %s, calling stop", config.controlURL, n.tsnetServer.ControlURL))
				go n.stop()
				// delete files?
				//n.createNode("")
			} else if config.hostname != n.tsnetServer.Hostname {
				// TODO How to use edit prefs?
				n.getLogger("setConfig").Debug(fmt.Sprintf("hostname changed %s -> %s, nothing for now", config.hostname, n.tsnetServer.Hostname))

			}
		}
	}
}

func (n *TSNetNode) createServer() error {
	n.servMux.Lock()
	defer n.servMux.Unlock()

	if n.tsnetServer != nil {
		err := errors.New("tsnet node already running")
		n.getLogger("createServer").Error(err)
		return err
	}
	s := new(tsnet.Server)

	s.Dir = n.tsnetDir
	s.Hostname = n.desiredConfig.hostname
	s.ControlURL = n.desiredConfig.controlURL
	s.AuthKey = n.desiredConfig.authKey

	s.Logf = nil
	s.UserLogf = func(format string, args ...any) {
		if !strings.Contains(format, "restart with TS_AUTHKEY set") {
			n.getLogger("createServer").AddNote("tsnet UserLogf").Log(fmt.Sprintf(format, args...))
		}
	}

	n.tsnetServer = s

	return nil
}

func (n *TSNetNode) startNode() error {
	logger := n.getLogger("startNode")

	n.nodeStatus.transitory = "connecting"
	go n.sendStatus()

	err := n.createServer()
	if err != nil {
		return err
	}

	lc, err := n.tsnetServer.LocalClient()
	if err != nil {
		logger.Clone().AddNote("LocalClient()").Error(err)
		return err
	}

	if len(n.desiredConfig.tags) != 0 {
		tags := make([]string, len(n.desiredConfig.tags))
		for i, t := range n.desiredConfig.tags {
			tags[i] = fmt.Sprintf("tag:%s", t)
		}
		maskedPrefs := ipn.MaskedPrefs{
			Prefs:            ipn.Prefs{AdvertiseTags: tags},
			AdvertiseTagsSet: true}
		lc.EditPrefs(context.Background(), &maskedPrefs)
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
				if !strings.Contains(err.Error(), "context canceled") {
					logger.Clone().AddNote("busWatcher.Next()").Error(err)
				}
				break
			}

			if newData.NetMap != nil {
				// note that netmap contains much more than peers!
				// it also contains UserProfiles:
				// fmt.Println("peers:")
				// for id, peer := range newData.NetMap.Peers {
				// 	fmt.Println(id, peer.ComputedNameWithHost(), "disp:", peer.DisplayName(false), "tags:", peer.Tags(), peer.User())
				// }
				// fmt.Println("user profiles:")
				// for id, p := range newData.NetMap.UserProfiles {
				// 	fmt.Println(id, p.DisplayName, p.LoginName)
				// }

				n.ingestPeers(lc, newData.NetMap.Peers)
				// send notification in goroutine
				go n.sendPeerUsersEvent()
			}

			lcStatus, err := lc.Status(context.Background())
			if err != nil {
				logger.Clone().AddNote("buswatcher lc.Status()").Error(err)
			}

			if n.nodeStatus.ingest(newData, lcStatus) {
				n.sendStatus()
			}

			n.startStopHTTPS()
		}
		err = busWatcher.Close()
		if err != nil {
			logger.Clone().AddNote("busWatcher.Close error").Error(err)
		}
	}()

	n.ln80, err = n.tsnetServer.Listen("tcp", ":80")
	if err != nil {
		logger.Clone().AddNote("Listen()").Error(err)
		return err
	}
	go n.handler(n.ln80)

	n.sendStatus()

	n.startStopHTTPS()

	return nil
}

func (n *TSNetNode) startStopHTTPS() {
	if n.ln443 == nil && n.nodeStatus.magicDNS && n.nodeStatus.httpsAvailable {
		var err error
		n.ln443, err = n.tsnetServer.ListenTLS("tcp", ":443")
		if err != nil {
			n.getLogger("startStopHTTPS ListenTLS()").Error(err)
			return
		}
		go n.handler(n.ln443)
		n.sendStatus()
	} else if n.ln443 != nil && (!n.nodeStatus.magicDNS || !n.nodeStatus.httpsAvailable) {
		err := n.ln443.Close()
		if err != nil {
			n.getLogger("startStopHTTPS ln443.Close()").Error(err)
		}
		n.ln443 = nil
		n.sendStatus()
	}
}

func (n *TSNetNode) handler(ln net.Listener) {
	err := http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggerFn := n.getLogger("http.Serve").Clone
		loggerFn().Debug("tsnet got request")

		status := n.getStatus()
		if !status.Usable {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		if n.hasAppspaceID {
			r = r.WithContext(domain.CtxWithAppspaceID(r.Context(), n.appspaceID))
		}

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
		fullID := fullIdentifier(who.UserProfile.ID.String(), n.userFacingControlURL())
		r = r.WithContext(domain.CtxWithTSNetUserID(r.Context(), fullID))

		n.Router.ServeHTTP(w, r)
	}))
	if err != http.ErrServerClosed { // it always returns an error
		n.getLogger("handler").AddNote("http.Serve()").Error(err) // "Error: tsnet: use of closed network connection" when shutting down the server.
		return
	}
}

func (n *TSNetNode) stop() {
	n.servMux.Lock()
	defer n.servMux.Unlock()

	if n.tsnetServer == nil {
		return
	}
	n.nodeStatus.transitory = "disconnecting"
	go n.sendStatus()

	n.ln80.Close()
	if n.ln443 != nil {
		n.ln443.Close()
		n.ln443 = nil
	}
	err := n.tsnetServer.Close()
	if err != nil {
		n.getLogger("stop Close()").Error(err)
	}
	if n.busWatcherCtxCancel != nil {
		n.busWatcherCtxCancel()
		n.busWatcherCtxCancel = nil
	}
	n.tsnetServer = nil
	n.nodeStatus.reset()
	go n.sendStatus()
}

// might need a context here that gets passed to WhoIsNodeKey
// and maybe check before each iteration of the loop.
func (n *TSNetNode) ingestPeers(lc *tailscale.LocalClient, peers []tailcfg.NodeView) {
	n.usersMux.Lock()
	defer n.usersMux.Unlock()
	n.users = make([]tsUser, 0) // ooff, we delete the original data? That means we lose the ability to know if it changed?

	loggerFn := n.getLogger("ingestPeers").Clone
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
			if userID != nv.User().String() {
				// that's something that I'd like to know about!
				// But what is the meaning of this??
				err = fmt.Errorf("who.UserProfile.ID.String() != nv.User().String(): %s != %s", userID, nv.User().String())
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
					n.users[i].ingest(nv, who, n.userFacingControlURL())
					break
				}
			}
			if !found {
				u = tsUser{}
				u.ingest(nv, who, n.userFacingControlURL())
				n.users = append(n.users, u)
			}
		}
	}

	// sort users,
	// and then sort their devices, online first, then anything so long as it's stable.

	// then fire event that says peers changed.

	// for i, u := range n.users {
	// 	fmt.Println("user", i, u.id, u.displayName, u.loginName, u.sharee)
	// 	fmt.Printf("user %d nodes: ", i)
	// 	for _, nd := range u.nodes {
	// 		fmt.Printf("(id:%s name:%s) ", nd.ID, nd.Name)
	// 	}
	// 	fmt.Print("\n")
	// }
}

func (n *TSNetNode) sendStatus() {
	n.TSNetStatusEvents.Send(n.getStatus())
}

func (n *TSNetNode) getStatus() domain.TSNetStatus {
	stat := n.nodeStatus.asDomain()
	stat.ListeningTLS = n.ln443 != nil
	stat.URL = n.buildURL()
	stat.ControlURL = n.userFacingControlURL()
	return stat
}

func (n *TSNetNode) buildURL() string {
	proto := "http"
	if n.ln443 != nil {
		proto = "https"
	}
	addr := n.nodeStatus.ip4
	if addr == "" {
		return ""
	}
	if n.nodeStatus.magicDNS {
		addr = strings.TrimRight(n.nodeStatus.dnsName, ".")
	}
	return fmt.Sprintf("%s://%s", proto, addr)
}

func (n *TSNetNode) sendPeerUsersEvent() {
	n.TSNetPeersEvents.Send()
}

func (n *TSNetNode) getPeerUsers() []domain.TSNetPeerUser {
	ret := make([]domain.TSNetPeerUser, len(n.users))
	for i, u := range n.users {
		ret[i] = u.asDomain()
	}
	return ret
}

func (n *TSNetNode) userFacingControlURL() string {
	if n.tsnetServer == nil {
		return ""
	}
	if n.tsnetServer.ControlURL == "" {
		return "tailscale.com"
	}
	return n.tsnetServer.ControlURL
}

func (n *TSNetNode) getLogger(note string) *record.DsLogger {
	r := record.NewDsLogger().AddNote("TSNetNode")
	if n.hasAppspaceID {
		r = r.AppspaceID(n.appspaceID)
	}
	if note != "" {
		r.AddNote(note)
	}
	return r
}

type tsNodeStatus struct {
	dnsName        string
	tailnet        string
	magicDNS       bool
	tags           []string
	keyExpiry      *time.Time
	httpsAvailable bool
	ip4            string
	ip6            string
	errMessage     string
	state          string
	browseToURL    string
	loginFinished  bool
	warnings       map[string]health.UnhealthyState
	transitory     string // "" or "connect" or "disconnect" ["deleting"?] indicates if server was commanded to start or to stop.
	// TODO add node expiry
}

// ingest note that any part of the status is non-nil only if it changed.
func (n *tsNodeStatus) ingest(data ipn.Notify, lcStatus *ipnstate.Status) (changed bool) {
	if n.ingestLCStatus(lcStatus) {
		changed = true
	}

	if data.NetMap != nil {
		changed = true                 // but maybe not?
		n.dnsName = data.NetMap.Name   // Name is "dns name with trailing dot"
		n.tailnet = data.NetMap.Domain // Domain is "tailnet name"
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
		if *data.State == ipn.Running || *data.State == ipn.NeedsLogin || *data.State == ipn.NeedsMachineAuth {
			n.transitory = ""
		}
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

func (n *tsNodeStatus) ingestLCStatus(lcStatus *ipnstate.Status) bool {
	if lcStatus == nil {
		return false
	}
	changed := false
	if lcStatus.CurrentTailnet != nil {
		magicDNS := lcStatus.CurrentTailnet.MagicDNSEnabled
		if n.magicDNS != magicDNS {
			n.magicDNS = magicDNS
			changed = true
		}
	}
	if lcStatus.Self != nil && lcStatus.Self.Tags != nil {
		tags := lcStatus.Self.Tags.AsSlice()
		if tags != nil {
			n.tags = tags
		}
	}
	if lcStatus.Self != nil {
		statusKeyExp := lcStatus.Self.KeyExpiry
		if statusKeyExp != nil {
			if n.keyExpiry == nil {
				n.keyExpiry = statusKeyExp
				changed = true
			} else if !n.keyExpiry.Equal(*statusKeyExp) {
				n.keyExpiry = statusKeyExp
				changed = true
			}
		} else if n.keyExpiry != nil { // if statusKeyExp is nil
			n.keyExpiry = nil
			changed = true
		}
	}

	return changed
}

func (n *tsNodeStatus) reset() {
	n.dnsName = ""
	n.tailnet = ""
	n.magicDNS = false
	n.keyExpiry = nil
	n.tags = []string{}
	n.httpsAvailable = false
	n.ip4 = ""
	n.ip6 = ""
	//n.errMessage = ""
	n.state = ""
	n.browseToURL = ""
	n.loginFinished = false
	//n.warnings = ""
	n.transitory = ""
}

func (n *tsNodeStatus) asDomain() domain.TSNetStatus {
	warnings := make(map[string]domain.TSNetWarning)
	for w, d := range n.warnings {
		warnings[w] = domain.TSNetWarning{
			Title:               d.Title,
			Text:                d.Text,
			Severity:            string(d.Severity),
			ImpactsConnectivity: d.ImpactsConnectivity,
		}
	}
	ret := domain.TSNetStatus{
		Tailnet:         n.tailnet,
		MagicDNSEnabled: n.magicDNS,
		KeyExpiry:       n.keyExpiry,
		Tags:            n.tags,
		Name:            n.dnsName,
		IP4:             n.ip4,
		IP6:             n.ip6,
		HTTPSAvailable:  n.httpsAvailable,
		ErrMessage:      n.errMessage,
		State:           n.state,
		Usable:          n.state == "Running" && len(n.tags) != 0,
		BrowseToURL:     n.browseToURL,
		LoginFinished:   n.loginFinished,
		Warnings:        warnings,
		Transitory:      n.transitory,
	}
	return ret
}

type tsUser struct {
	id          string // includes the "userid:..." prefix.
	controlURL  string
	displayName string
	loginName   string
	picURL      string
	sharee      bool
	nodes       []domain.TSNetUserDevice
}

func (u *tsUser) ingest(nv tailcfg.NodeView, who *apitype.WhoIsResponse, controlURL string) {
	u.id = who.UserProfile.ID.String()
	u.controlURL = controlURL
	u.displayName = who.UserProfile.DisplayName
	u.loginName = who.UserProfile.LoginName
	u.picURL = who.UserProfile.ProfilePicURL
	u.sharee = nv.Hostinfo().ShareeNode()

	if u.nodes == nil {
		u.nodes = make([]domain.TSNetUserDevice, 0)
	}
	u.nodes = append(u.nodes, ingestNode(nv))
}

func ingestNode(nv tailcfg.NodeView) domain.TSNetUserDevice {
	return domain.TSNetUserDevice{
		ID:          string(nv.StableID()),
		Name:        nv.Name(),
		Online:      nv.Online().Clone(),
		LastSeen:    nv.LastSeen().Clone(),
		OS:          nv.Hostinfo().OS(), // lots more to do here
		DeviceModel: nv.Hostinfo().DeviceModel(),
		App:         nv.Hostinfo().App(),
	}
}

func (u *tsUser) asDomain() domain.TSNetPeerUser {
	id := strings.TrimPrefix(u.id, "userid:")
	return domain.TSNetPeerUser{
		ID:          id,
		ControlURL:  u.controlURL,
		LoginName:   u.loginName,
		DisplayName: u.displayName,
		Sharee:      u.sharee,
		Devices:     u.nodes, // do we need to clone that? NO, I don't think so,
		FullID:      fullIdentifier(id, u.controlURL),
	}
}

func fullIdentifier(id, controlURL string) string {
	id = strings.TrimPrefix(id, "userid:")
	if controlURL == "" {
		controlURL = "tailscale.com"
	}
	return fmt.Sprintf("%s@%s", id, controlURL)
}
