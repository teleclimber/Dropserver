package server

import (
// 	"fmt"
 	"testing"
// 	"net/http"
// 	"net/http/httptest"
// 	"github.com/golang/mock/gomock"
// 	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)


func TestGetSubdomains(t *testing.T) {
	cases := []struct {
		input    string
		subdomains []string
		ok       bool
	}{
		{"dropserver.develop", []string{}, true},
		{"dropserver.xyz", []string{}, false},
		{"dropserver", []string{}, false},
		{"du-report.dropserver.develop", []string{"du-report"}, true},
		{"foo.du-report.dropserver.develop", []string{"foo","du-report"}, true},
		{"foo.du-report.dropserver.develop:3000", []string{"foo","du-report"}, true},
	}

	// TODO: cases should take configuration into consideration
	// ..config needs to be set up and injectable.

	for _, c := range cases {
		subdomains, ok := getSubdomains(c.input)
		if c.ok != ok {
			t.Errorf("%s: expected OK %t, got %t", c.input, c.ok, ok)
		}
		if !stringSlicesEqual(c.subdomains, subdomains) {
			t.Error(c.input, "expected subdomains / got:", c.subdomains, subdomains)
		}
	}
}

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
// from https://yourbasic.org/golang/compare-slices/
func stringSlicesEqual(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    for i, v := range a {
        if v != b[i] {
            return false
        }
    }
    return true
}

// // TestServerProxy tests the proxy (sort of, we're just trying it out here)
// func TestServerProxy(t *testing.T) {
// 	// Testing this function is super non-ideal because
// 	// - the handler function in Server is welded to the Server, so have to create the server.
// 	// - the proxy handler performs multiple things (determine app-space and app, get sandbox, proxy)
// 	//   ..these should all be composable middlewares
// 	// - the proxy will forward to an address, so gotta set a server up to receive

// 	mockCtrl := gomock.NewController(t)
//     defer mockCtrl.Finish()

// 	sM := domain.NewMockSandboxManagerI(mockCtrl)
// 	sandbox := domain.NewMockSandboxI(mockCtrl)
// 	metrics := domain.NewMockMetricsI(mockCtrl)
// 	logClient := domain.NewMockLogCLientI(mockCtrl)

// 	hostAppSpace := map[string]string{
// 		"as1.teleclimber.dropserver.org": "as1"}

// 	appSpaceApp := map[string]string{
// 		"as1": "app1"}

// 	server := &Server{
// 		SandboxManager: sM,
// 		Metrics: metrics,
// 		HostAppSpace: &hostAppSpace,
// 		AppSpaceApp: &appSpaceApp}

// 	// dummy server to stand in for sandbox
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		fmt.Println("got request in dummy sandbox server")
// 		w.WriteHeader(200)
// 		fmt.Fprintf(w, "Hello World")
// 	}))
// 	defer ts.Close()

// 	sM.EXPECT().GetForAppSpace("app1", "as1").DoAndReturn( func(a, b string) chan domain.SandboxI {
// 		sandboxChan := make( chan domain.SandboxI )
// 		go func() {
// 			sandboxChan <- sandbox
// 		}()
// 		return sandboxChan
// 	})

// 	sandbox.EXPECT().GetName().Return("1")
// 	sandbox.EXPECT().GetAddress().Return(ts.URL)
// 	sandbox.EXPECT().GetTransport().Return(http.DefaultTransport)

// 	taskCh := make(chan bool)
// 	sandbox.EXPECT().TaskBegin().Return(taskCh)
// 	go func() {
// 		<- taskCh
// 		fmt.Println("task done")
// 	}()

// 	sandbox.EXPECT().GetLogClient().Return(logClient)
// 	logClient.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any())
// 	metrics.EXPECT().HostHandleReq(gomock.Any())

// 	// from https://blog.questionable.services/article/testing-http-handlers-go/
// 	// craft a request
// 	req, err := http.NewRequest("GET", "/", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req.Host = "as1.teleclimber.dropserver.org"

// 	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(server.handleRequest)

// 	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method 
// 	// directly and pass in our Request and ResponseRecorder.
// 	handler.ServeHTTP(rr, req)

// 	// Check the status code is what we expect.
// 	if status := rr.Code; status != http.StatusOK {
// 		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
// 	}

// 	// Check the response body is what we expect.
// 	expected := `Hello World`
// 	if rr.Body.String() != expected {
// 		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
// 	}
// }


