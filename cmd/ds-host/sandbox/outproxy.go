package sandbox

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-host/runtimeconfig"
)

type OutProxy struct {
	Log    func(string)
	server *http.Server
	port   int
}

func (o *OutProxy) Start(config domain.RuntimeConfig, mitm bool) error {

	proxy := goproxy.NewProxyHttpServer()

	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			o.Log(fmt.Sprintf("Request: %s %s %s %s", req.Proto, req.Method, req.Host, req.URL))
			return req, nil
		})
	proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		o.Log(fmt.Sprintf("TLS Connect to: %s", host))
		return goproxy.OkConnect, host
	})

	if mitm { //not used yet
		proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	}

	s := runtimeconfig.GetSSRFGuardian(config)

	dialer := &net.Dialer{
		Control: s.Safe,
	}
	proxy.Tr.DialContext = dialer.DialContext

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	o.port = listener.Addr().(*net.TCPAddr).Port

	o.server = &http.Server{
		Handler: proxy,
	}

	go func() {
		err := o.server.Serve(listener)
		if err != nil && err != http.ErrServerClosed {
			o.Log(fmt.Sprintf("error from outproxy server: %s", err.Error()))
		}
	}()

	return nil
}

func (o *OutProxy) Port() int {
	return o.port
}

func (o *OutProxy) Stop() {
	if o.server != nil {
		o.server.Shutdown(context.TODO())
	}
}
