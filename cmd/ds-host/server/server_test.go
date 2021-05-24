package server

// TODO Frontend is no longer in server. Move this tedt to user routes.
// func TestFrontend(t *testing.T) {
// 	rtc := domain.RuntimeConfig{}
// 	rtc.Exec.UserRoutesDomain = "user.routes.com"
// 	rtc.Server.NoSsl = true

// 	mockCtrl := gomock.NewController(t)
// 	defer mockCtrl.Finish()

// 	s := &Server{
// 		Config: &rtc,
// 	}
// 	s.Init()

// 	dirEntries, err := dshostfrontend.FS.ReadDir("dist/frontend-assets/js")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	targetFile := ""
// 	for _, entry := range dirEntries {
// 		if entry.IsDir() {
// 			return
// 		}
// 		targetFile = entry.Name()
// 	}
// 	if targetFile == "" {
// 		t.Error("failed to find a JS file to test frontend server. Please build frontend first. Sorry for the mad coupling.")
// 	}

// 	testServer := httptest.NewServer(s.mux)
// 	defer testServer.Close()

// 	req, err := http.NewRequest("GET", testServer.URL+"/frontend-assets/js/"+targetFile, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req.Host = rtc.Exec.UserRoutesDomain

// 	client := &http.Client{}
// 	res, err := client.Do(req)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	if res.StatusCode != http.StatusOK {
// 		t.Fatal("expected status 200, got " + res.Status)
// 	}

// 	res.Body.Close()

// 	// TODO: this test is wrong. It gets a 200 OK because it receives a directory listing.
// 	// But we want to kill dir listings for frontend assets.

// 	// body, err := io.ReadAll(res.Body)
// 	// if err != nil {
// 	// 	t.Fatal(err)
// 	// }

// 	// bod := string(body)
// 	// if !strings.Contains(bod, "<!DOCTYPE html>") {
// 	// 	t.Fatal("expected index html")
// 	// }
// }
