package userroutes

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestExtractFiles(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	part, err := writer.CreateFormFile("app_dir", "foo.txt")
	if err != nil {
		panic(err)
	}
	fakeFoo := newFakeFile(200)
	if _, err = io.Copy(part, fakeFoo); err != nil {
		panic(err)
	}

	// another file
	part, err = writer.CreateFormFile("app_dir", "bar.txt")
	if err != nil {
		panic(err)
	}
	fakeBar := newFakeFile(1000)
	if _, err = io.Copy(part, fakeBar); err != nil {
		panic(err)
	}

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
	fileData := appRoutes.extractFiles(req)

	if len(*fileData) != 2 {
		t.Error("fileData should have 2 files", fileData)
	} else {
		if !fakeFoo.matches((*fileData)["foo.txt"]) || !fakeBar.matches((*fileData)["bar.txt"]) {
			t.Error("filedata does not match", fileData)
		}
	}

}

// Test that extract files doesn't fail with empty body
func TestExtractFilesEmptyBody(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	err := writer.Close()
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest(http.MethodPost, "/", buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	appRoutes := &ApplicationRoutes{}
	fileData := appRoutes.extractFiles(req)

	if len(*fileData) != 0 {
		t.Error("filedata shouldbe zero length", fileData)
	}

}

// check taht a file with no content doesn't muck things up.
func TestExtractFilesEmptyFile(t *testing.T) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	_, err := writer.CreateFormFile("app_dir", "foo.txt")
	if err != nil {
		panic(err)
	}

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
	fileData := appRoutes.extractFiles(req)

	if len(*fileData) != 1 {
		t.Error("filedata should be have 1 file", fileData)
	} else if len((*fileData)["foo.txt"]) != 0 {
		t.Error("length of file should be 0", fileData)
	}
}

// from https://play.golang.org/p/9BbS54d8pb
// and https://stackoverflow.com/questions/28174970/implementing-reader-interface

type fakeFile struct {
	// stash supposed file size and currently read amount
	size int
	read int
}

func newFakeFile(size int) *fakeFile { // need to pass size I suppose
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

// matches compares the bytes from the args
// with the bytes that would be produced by fake file
func (f *fakeFile) matches(b []byte) bool {
	if f.size != len(b) {
		return false
	}

	theByte := []byte("A")[0]
	for i := 0; i < f.size; i++ {
		if b[i] != theByte {
			return false
		}
	}
	return true
}
