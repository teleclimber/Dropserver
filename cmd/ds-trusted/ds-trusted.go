package main

import (
	"net"
	"net/http"
	"net/rpc"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
	"github.com/teleclimber/DropServer/cmd/ds-trusted/appfiles"
	"github.com/teleclimber/DropServer/cmd/ds-trusted/record"
	"github.com/teleclimber/DropServer/cmd/ds-trusted/runtimeconfig"
	"github.com/teleclimber/DropServer/cmd/ds-trusted/trustedservice"
)

// what do we do in here?
// start a server that ds-host will forward connections to.
// In here we do:
// - create and delete sandbox data dirs
// - create and manage app-spaces and app code
// - perform the operations on app-space data that do not require custom code
//   ..like crud on DB

// For now the main function is to manage app-spaces and app code. Meaning:
// - create new app-space dirs with appropriate permissions
// - create new app code dir, with appropriate permissions, (owner, group?)
// -

// We should have injectable packages for
// - logging and metrics
// - config

func main() {
	runtimeConfig := runtimeconfig.Load()

	logger := record.NewLogClient(runtimeConfig)

	logger.Log(domain.INFO, nil, "ds-trusted starting up")

	appFiles := appfiles.AppFiles{
		Logger: logger}

	trustedAPI := trustedservice.TrustedAPI{
		AppFiles: &appFiles}

	rpc.Register(&trustedAPI)
	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":1234") // this should be a config I suppose, but not clear how it makes it into ds-trusted
	if err != nil {
		panic(err)
	}

	http.Serve(listener, nil)
	// ^^ blocks, whihc is fine until we need to do something more elaborate
}
