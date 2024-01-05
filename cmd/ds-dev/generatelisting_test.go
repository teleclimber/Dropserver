package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestGetPackageFile(t *testing.T) {
	files := fileList{
		{"abc.txt", []byte("hello")},
		{"deep/nested/file.txt", []byte("it's dark down here")},
	}
	_, err := getPackageFile(createPackage(files), "dropapp.json")
	if err == nil {
		t.Error("expected error")
	}
	data, err := getPackageFile(createPackage(files), "abc.txt")
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(files[0].contents, data) {
		t.Errorf("bytes not equal %v %v", files[0].contents, data)
	}
}

type fileList []struct {
	name     string
	contents []byte
}

func createPackage(files fileList) io.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, f := range files {
		hdr := &tar.Header{
			Name: f.name,
			Mode: 0644,
			Size: int64(len(f.contents))}
		err := tw.WriteHeader(hdr)
		if err != nil {
			panic(err)
		}
		_, err = tw.Write(f.contents)
		if err != nil {
			panic(err)
		}
	}
	tw.Close()

	var outBuf bytes.Buffer
	gzw := gzip.NewWriter(&outBuf)
	// gzw.Name = name
	// gzw.Comment = comment
	// gzw.ModTime = modTime.Round(time.Second)

	_, err := gzw.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
	gzw.Close()

	return &outBuf
}
