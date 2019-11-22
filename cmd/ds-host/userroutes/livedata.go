package userroutes

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/internal/shiftpath"
)

// consts borrowed from gorilla/websckets chat example
// https://github.com/gorilla/websocket/blob/master/examples/chat/client.go

type websocketConstants struct {
	// Time allowed to write a message to the peer.
	writeWait time.Duration

	// Time allowed to read the next pong message from the peer.
	pongWait time.Duration

	// Maximum message size allowed from peer.
	maxMessageSize int64
}

// Send pings to peer with this period. Must be less than pongWait.
func (c *websocketConstants) pingPeriod() time.Duration {
	return (c.pongWait * 9) / 10
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, // come up with more appropriate size
	WriteBufferSize: 1024,
}

type token struct {
	userID domain.UserID
	expire time.Time
}

// LiveDataRoutes provides live data update service.
type LiveDataRoutes struct {
	Authenticator     domain.Authenticator
	JobController     domain.MigrationJobController
	MigrationJobModel domain.MigrationJobModel
	Logger            domain.LogCLientI
	wsConsts          *websocketConstants
	tokens            map[string]token
	tokenMux          sync.Mutex
	tokenExp          time.Duration
	tokenTicker       *time.Ticker
	tokenTickerDone   chan struct{}
	clients           map[string]*liveDataClient
	clientMux         sync.Mutex
	clientClosed      chan string
	stop              bool
}

// Init makes the maps and what not
func (l *LiveDataRoutes) Init() {
	l.wsConsts = &websocketConstants{
		writeWait:      10 * time.Second,
		pongWait:       60 * time.Second,
		maxMessageSize: 512}
	l.tokens = make(map[string]token)
	l.clients = make(map[string]*liveDataClient)
	l.clientClosed = make(chan string)
	l.tokenExp = time.Second * 30
	l.tokenTicker = time.NewTicker(time.Minute)
	l.tokenTickerDone = make(chan struct{})
	go l.tokenCleanup()
}

// Stop asks all conns to hangup and returns when they did
func (l *LiveDataRoutes) Stop() {
	close(l.tokenTickerDone)
	l.tokenTicker.Stop()

	l.clientMux.Lock()
	l.stop = true

	numClient := len(l.clients)

	for _, c := range l.clients {
		c.stop()
	}
	l.clientMux.Unlock()

	if numClient > 0 {
		for range l.clientClosed {
			numClient--
			if numClient == 0 {
				break
			}
		}
	}
}

func (l *LiveDataRoutes) tokenCleanup() {
	//l.tokenTicker = time.NewTicker(time.Minute)
	defer l.tokenTicker.Stop()
	for {
		select {
		case <-l.tokenTicker.C:
			now := time.Now()
			l.tokenMux.Lock()
			for tokStr, t := range l.tokens {
				if t.expire.Before(now) {
					delete(l.tokens, tokStr)
				}
			}
			l.tokenMux.Unlock()
		case <-l.tokenTickerDone:
			return
		}
	}
}

// ServeHTTP handles traffic for live data
// It will create a short term token for the ws connection to use
// And it will begin the live data channel when it's called again with the token
// -> although I may not have fully understood the thing about cookies not being sent to ws requests??
func (l *LiveDataRoutes) ServeHTTP(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	// what's our route look like?
	// ..org/live-data/ => request token
	// ..org/live-data/some-token => start in earnest
	head, tail := shiftpath.ShiftPath(routeData.URLTail)
	routeData.URLTail = tail
	switch head {
	case "":
		l.serveToken(res, req, routeData)
	default:
		l.startWsConn(res, req, head)
	}
}

func (l *LiveDataRoutes) serveToken(res http.ResponseWriter, req *http.Request, routeData *domain.AppspaceRouteData) {
	dsErr := l.Authenticator.AccountAuthorized(res, req, routeData)
	if dsErr != nil {
		res.WriteHeader(http.StatusUnauthorized)
		return
	}

	l.tokenMux.Lock()
	defer l.tokenMux.Unlock()
	tokStr := randomString()
	_, ok := l.tokens[tokStr]
	if ok {
		http.Error(res, "duplicate token string! Buy a lotto ticket NOW", http.StatusInternalServerError)
		return
	}

	l.tokens[tokStr] = token{routeData.Cookie.UserID, time.Now().Add(l.tokenExp)}

	resp := GetStartLiveDataResp{Token: tokStr}
	writeJSON(res, resp)
}

func (l *LiveDataRoutes) startWsConn(res http.ResponseWriter, req *http.Request, tokStr string) {
	l.tokenMux.Lock()
	tok, ok := l.tokens[tokStr]
	delete(l.tokens, tokStr)
	l.tokenMux.Unlock()
	if !ok || tok.expire.Before(time.Now()) {
		http.Error(res, "token not found", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(res, req, nil)
	if err != nil {
		l.Logger.Log(domain.WARN, nil, "unable to upgrade to websocket conn "+err.Error()+" ..req origin: "+req.Header.Get("Origin"))
		http.Error(res, "unable to upgrade to websocket", http.StatusInternalServerError)
		return
	}

	client := &liveDataClient{
		userID:            tok.userID,
		requestID:         tokStr,
		conn:              conn,
		wsConsts:          l.wsConsts,
		jobController:     l.JobController,
		migrationJobModel: l.MigrationJobModel,
		logger:            l.Logger}
	client.start()
	client.subscribeJobs()

	l.clientMux.Lock()
	l.clients[tokStr] = client
	l.clientMux.Unlock()
	go func() {
		<-client.done
		l.clientMux.Lock()
		delete(l.clients, tokStr)
		l.clientMux.Unlock()
		l.clientClosed <- tokStr
	}()
}

type updateData struct {
	migrationJob *domain.MigrationJob
	statusData   domain.MigrationStatusData
}

// I think a separate live-data client is the right appraoch
// the live client can read commands from remote and [un]subscribe to events as needed
// It pipes these subscriptions down to remote.
type liveDataClient struct {
	userID    domain.UserID
	requestID string
	conn      *websocket.Conn
	wsConsts  *websocketConstants

	ticker *time.Ticker

	jobController     domain.MigrationJobController
	migrationJobModel domain.MigrationJobModel

	updatesPipe chan updateData

	stopWrites chan struct{}
	done       chan struct{}
	stopped    bool
	stopMux    sync.Mutex

	logger domain.LogCLientI
}

func (c *liveDataClient) start() {
	c.stopWrites = make(chan struct{})
	c.done = make(chan struct{})
	c.ticker = time.NewTicker(c.wsConsts.pingPeriod())

	//c.jobsPipe = make(chan []*domain.MigrationJob)
	c.updatesPipe = make(chan updateData)

	go c.writePump()
	go c.readPump()
}

// stop stops the client.
// It can be called multiple times safely.
func (c *liveDataClient) stop() { // I suppose?
	c.stopMux.Lock()
	defer c.stopMux.Unlock()
	if !c.stopped {
		c.jobController.UnsubscribeOwner(c.userID, c.requestID)
		c.ticker.Stop()
		c.conn.Close()
		close(c.stopWrites)
		close(c.done)
		c.stopped = true
	}
}

func (c *liveDataClient) subscribeJobs() {
	updateChan, curStat := c.jobController.SubscribeOwner(c.userID, c.requestID)

	for _, cs := range curStat {
		job, err := c.migrationJobModel.GetJob(cs.JobID)
		if err != nil {
			c.logger.Log(domain.ERROR, nil, "error getting job in livedataclient subscribe "+err.ExtraMessage())
			continue // frontend will have trouble with subsequent updates, but oh well.
		}
		c.updatesPipe <- updateData{
			migrationJob: job,
			statusData:   cs,
		}
	}

	go func() {
		for update := range updateChan {
			var job *domain.MigrationJob
			var err domain.Error
			if update.Status == domain.MigrationStarted {
				job, err = c.migrationJobModel.GetJob(update.JobID)
				if err != nil {
					c.logger.Log(domain.ERROR, nil, "error getting job in livedataclient subscribe "+err.ExtraMessage())
					//continue // frontend will have trouble with subsequent updates, but oh well.
				}
			}
			c.updatesPipe <- updateData{job, update}
		}
	}()
}

// what are our possible subscriptions for now?
// - job status for account
// - job status for appspace?
// Later:
// - sandbox state for each appspace
// - appspace usage
type command struct {
	Name       string            `json:"name"`
	AppspaceID domain.AppspaceID `json:"appspace_id"` // ignore if empty or whatever?
}

func (c *liveDataClient) readPump() {
	pongWait := c.wsConsts.pongWait
	c.conn.SetReadLimit(c.wsConsts.maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		fmt.Println("readpump PongHandler")
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			fmt.Println("error setting read deadline in SetPongHandler")
		}
		return err
	})
	for {
		_, _, err := c.conn.ReadMessage()
		fmt.Println("readpump ReadMessage")
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Log(domain.DEBUG, nil, "IsUnexpectedCloseError() "+err.Error())
			}
			fmt.Println("readPump ReadMessage error", err.Error())
			break
		}
		// message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		// c.hub.broadcast <- message
	}
	// for {
	// 	var cmd command
	// 	err := c.conn.ReadJSON(&cmd)
	// 	if err != nil {
	// 		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
	// 			c.logger.Log(domain.DEBUG, nil, "IsUnexpectedCloseError() "+err.Error())
	// 		} else {
	// 			c.logger.Log(domain.ERROR, nil, "unexpected ReadJSON error "+err.Error())
	// 		}
	// 		break
	// 	}

	// 	// do something with command
	// }
	// ^^ this is erroring on ReadJSON."io: read/write on closed pipe" Not sure why.
}

// writePump takes incoming messages from sources we are subscribed to
// and sends them out to the remote.
func (c *liveDataClient) writePump() {
	var err error
	var errStr string
	defer c.stop()
	defer func() {
		if err != nil {
			c.logger.Log(domain.ERROR, nil, "writePump error: "+errStr)
			// We may want to limit the errors we log, because we may get too many remote disconnects clogging up the log
		}
	}()
	for {
		select {
		case update := <-c.updatesPipe:
			data := newUpdateData(update)
			err = c.conn.SetWriteDeadline(time.Now().Add(c.wsConsts.writeWait))
			if err != nil {
				errStr = "updatesPipe SetWriteDeadline " + err.Error()
				return
			}
			err = c.conn.WriteJSON(data)
			if err != nil {
				errStr = "updatesPipe WriteJSON " + err.Error()
				return
			}
		case <-c.ticker.C:
			fmt.Println("c.ticker.C triggered")

			err = c.conn.SetWriteDeadline(time.Now().Add(c.wsConsts.writeWait))
			if err != nil {
				errStr = "ticker SetWriteDeadline " + err.Error()
				return
			}
			err = c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				errStr = "ticker WriteMessage " + err.Error()
				return
			}
		case <-c.stopWrites:
			return
		}
	}
}

// converting types

// NewMigrationJobResp converts a domain job to a response type ... :/
func newMigrationJobResp(j *domain.MigrationJob) *MigrationJobResp {
	if j == nil {
		return nil
	}
	return &MigrationJobResp{
		JobID:      j.JobID,
		OwnerID:    j.OwnerID,
		AppspaceID: j.AppspaceID,
		ToVersion:  j.ToVersion,
		Created:    j.Created,
		Finished:   j.Finished,
		Priority:   j.Priority,
		Error:      j.Error}
}

func newUpdateData(update updateData) (r MigrationStatusResp) {
	r.JobID = update.statusData.JobID
	r.MigrationJob = newMigrationJobResp(update.migrationJob)
	r.Status = getStatusString(update.statusData.Status)
	r.Started = update.statusData.Started
	r.Finished = update.statusData.Finished
	r.Error = update.statusData.ErrString
	r.CurSchema = update.statusData.CurSchema
	return
}

func getStatusString(s domain.MigrationJobStatus) string {
	switch s {
	case domain.MigrationStarted:
		return "started"
	case domain.MigrationRunning:
		return "running"
	case domain.MigrationFinished:
		return "finished"
	default:
		panic("Missing case for MigrationJobStatus")
	}
}

////////////
// random string stuff
const chars61 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand2 = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomString() string {
	b := make([]byte, 24)
	for i := range b {
		b[i] = chars61[seededRand2.Intn(len(chars61))]
	}
	return string(b)
}
