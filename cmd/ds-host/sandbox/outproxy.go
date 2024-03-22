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
	AppspaceID domain.AppspaceID
	server     *http.Server
}

func (o *OutProxy) Start(config domain.RuntimeConfig) int {

	proxy := goproxy.NewProxyHttpServer()
	//proxy.Verbose = true

	proxy.OnRequest().DoFunc(
		func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			// for now just output the URL:
			fmt.Println(r.URL)
			return r, nil
		})
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	s := runtimeconfig.GetSSRFGuardian(config)

	dialer := &net.Dialer{
		Control: s.Safe,
	}
	proxy.Tr.DialContext = dialer.DialContext

	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	o.server = &http.Server{
		Handler: proxy,
	}

	go func() {
		err := o.server.Serve(listener)
		fmt.Println(err)
	}()

	return listener.Addr().(*net.TCPAddr).Port
}

func (o *OutProxy) Stop() {
	if o.server != nil {
		o.server.Shutdown(context.TODO())
	}
}
