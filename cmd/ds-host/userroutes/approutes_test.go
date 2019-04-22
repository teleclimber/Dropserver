package userroutes

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestPost(t *testing.T) {
	fmt.Println("Testing post")

	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	//defer writer.Close()

	part, err := writer.CreateFormFile("app_dir", "foo.txt")
	if err != nil {
		panic(err)
	}
	fakeFile := newFakeFile(200)
	if _, err = io.Copy(part, fakeFile); err != nil {
		panic(err)
	}

	// another file?
	part, err = writer.CreateFormFile("app_dir", "bar.txt")
	if err != nil {
		panic(err)
	}
	fakeFile = newFakeFile(1000)
	if _, err = io.Copy(part, fakeFile); err != nil {
		panic(err)
	}

	// hmm, ok we don't have a url per se.
	// need to craft our request by hand.
	//http.Post(url, writer.FormDataContentType(), buf)

	err = writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	appRoutes := &ApplicationRoutes{}
	appRoutes.post(req, &domain.AppspaceRouteData{})

	//TODO: actually test that we got what we wanted.

}

// fakefile so we have something to upload

// from https://play.golang.org/p/9BbS54d8pb
// and https://stackoverflow.com/questions/28174970/implementing-reader-interface

type fakeFile struct {
	// stash supposed file size and currently read amount
	size int64
	read int64
}

func newFakeFile(size int64) *fakeFile { // need to pass size I suppose
	return &fakeFile{
		size: size,
		read: 0}
}

func (f *fakeFile) eof() bool {
	return f.read >= f.size
}

func (f *fakeFile) Read(p []byte) (n int, err error) {
	if f.eof() {
		err = io.EOF
		return
	}
	if l := len(p); l > 0 {
		for n < l {
			p[n] = []byte("A")[0]
			f.read++
			n++
			if f.eof() {
				break
			}
		}
	}
	return
}
