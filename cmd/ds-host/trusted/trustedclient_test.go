package trusted

// import (
// 	"net"
// 	"net/http"
// 	"net/rpc"
// 	"testing"

// 	"github.com/golang/mock/gomock"
// 	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
// )

// ////
// // hmm, do we really want tests to involve net/rpc?
// // It seems it would be better to test up to the client,
// // and test from the service out,
// // ..but leave rpc out of tests.
// // Or you need to make rpc injectable.

// func TestCall(t *testing.T) {
// 	mockCtl := gomock.NewController(t)
// 	defer mockCtl.Finish()

// 	// do we need to create an entire server?
// 	// > one way or another I'd like to test this without involving containers,
// 	// ..so yes.

// 	mockService(mockCtl)

// 	client, err := rpc.DialHTTP("tcp", ":1234")
// 	if err != nil {
// 		panic(err)
// 	}

// 	rpcClient := RPCClient{
// 		client: client}

// 	args := &domain.RPCArgs{A: 2, B: 3} <--- no longer part of domain
// 	var reply int

// 	err = rpcClient.call("Multiply", args, &reply)
// 	if err != nil {
// 		panic(err)
// 	}

// 	if reply != 6 {
// 		t.Error("Result not 6")
// 	}
// }

// func mockService(mockCtl *gomock.Controller) {
// 	mockTrustedService := domain.NewMockTrustedServiceI(mockCtl)

// 	type TrustedAPI int

// 	rpc.Register(mockTrustedService)
// 	rpc.HandleHTTP()
// 	listener, err := net.Listen("tcp", ":1234")
// 	if err != nil {
// 		panic(err)
// 	}

// 	go http.Serve(listener, nil)
// }


