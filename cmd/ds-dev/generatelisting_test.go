package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
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

func TestValidateVersionSequence(t *testing.T) {
	versions := setVersionData(map[domain.Version]versionData{
		domain.Version("2.0.0"): {schema: 2},
		domain.Version("1.0.0"): {schema: 1},
		domain.Version("3.0.0"): {schema: 2},
	})
	err := validateVersionSequence(versions)
	if err != nil {
		t.Error(err)
	}

	vData := versions[domain.Version("2.0.0")]
	vData.schema = 4
	versions[domain.Version("2.0.0")] = vData
	err = validateVersionSequence(versions)
	if err == nil {
		t.Error("expected error")
	}
}

func setVersionData(versions map[domain.Version]versionData) map[domain.Version]versionData {
	for v, d := range versions {
		d.version = v
		versions[v] = d
	}
	return versions
}
