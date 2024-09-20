package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestGetFileList(t *testing.T) {
	dir, err := makeTestDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	manifestPath := filepath.Join(dir, "dropapp.json")
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	f1Path := filepath.Join(dir, "file.txt")
	f1Info, err := os.Stat(f1Path)
	if err != nil {
		t.Fatal(err)
	}

	f2Path := filepath.Join(dir, "subdir", "file2.txt")
	f2Info, err := os.Stat(f2Path)
	if err != nil {
		t.Fatal(err)
	}

	expected := []FileListFile{
		{Name: ".git", IsDir: true, Ignore: true},
		{Name: "dropapp.json", IsDir: false, Size: manifestInfo.Size(), ModTime: manifestInfo.ModTime(), Ignore: false},
		{Name: "file.txt", IsDir: false, Size: f1Info.Size(), ModTime: f1Info.ModTime(), Ignore: false},
		{Name: "subdir/file2.txt", IsDir: false, Size: f2Info.Size(), ModTime: f2Info.ModTime(), Ignore: false},
	}

	dir = filepath.Clean(dir)

	list, err := GetFileList(dir)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(list, expected) {
		t.Error("got wrong FileList: ", list)
	}

	list, err = GetFileList(dir + "/")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(list, expected) {
		t.Error("got wrong FileList WITH TRAILING SLASH in dir: ", list)
	}
}

func TestTarFiles(t *testing.T) {
	dir, err := makeTestDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	list, err := GetFileList(dir)
	if err != nil {
		t.Error(err)
	}

	manifestBytes := []byte("{\"testing\":true}")
	var buf bytes.Buffer
	err = tarFiles(&buf, dir, list, manifestBytes)
	if err != nil {
		t.Error(err)
	}

	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			t.Fatal(err)
		}

		isManifest := false
		if hdr.Name == "dropapp.json" {
			isManifest = true
		}

		if hdr.Name != "file.txt" && hdr.Name != "subdir/file2.txt" && !isManifest {
			t.Error("got unexpected file: " + hdr.Name)
		}

		p := filepath.Join(dir, hdr.Name)
		info, err := os.Stat(p)
		if err != nil {
			t.Error(err)
		}
		size := info.Size()
		if isManifest {
			size = int64(len(manifestBytes))
		}
		if size != hdr.Size {
			t.Errorf("Wrong Size for %s: %v %v", hdr.Name, size, hdr.Size)
		}
		modTime := info.ModTime().Round(time.Second)
		if modTime != hdr.ModTime {
			t.Errorf("Wrong ModTime for %s: %v %v", hdr.Name, info.ModTime(), hdr.ModTime)
		}
		if hdr.FileInfo().Mode() != 0644 {
			t.Errorf("Wrong mode for %s: %v", hdr.Name, hdr.FileInfo().Mode())
		}

		var tarContents bytes.Buffer
		if _, err := io.Copy(&tarContents, tr); err != nil {
			t.Error(err)
		}
		var expectedContents []byte
		if expectedContents, err = os.ReadFile(p); err != nil {
			t.Error(err)
		}
		if isManifest {
			expectedContents = manifestBytes
		}
		if !bytes.Equal(tarContents.Bytes(), expectedContents) {
			t.Errorf("got different contenst for file %s", hdr.Name)
		}
	}
}

func TestGetAppFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	base := "some-file"
	_, err = getAppFile(dir, base)
	if err != nil {
		t.Fatal(err)
	}
	_, err = getAppFile(dir, base)
	if err == nil {
		t.Error("expected an error because trying to create the same file twice.")
	}
}

func makeTestDir() (string, error) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	mPath := filepath.Join(dir, "dropapp.json")
	if err := os.WriteFile(mPath, []byte("{}"), 0666); err != nil {
		return "", err
	}
	f1Path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f1Path, []byte("abc"), 0666); err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Join(dir, ".git"), 0777); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "git-file"), []byte("git-data"), 0666); err != nil {
		return "", err
	}

	f2Path := filepath.Join(dir, "subdir", "file2.txt")
	if err = os.MkdirAll(filepath.Dir(f2Path), 0777); err != nil {
		return "", err
	}
	if err = os.WriteFile(f2Path, []byte("abc"), 0666); err != nil {
		return "", err
	}
	return dir, nil
}

func TestGzipArchive(t *testing.T) {
	data := []byte("abc")
	var gzData bytes.Buffer
	modTime := time.Now()
	err := gzipArchive(&gzData, data, "test-name", "test-comment", modTime)
	if err != nil {
		t.Error(err)
	}

	gzr, err := gzip.NewReader(&gzData)
	if err != nil {
		t.Error(err)
	}

	if gzr.Header.Name != "test-name" {
		t.Error("got wrong Name", gzr.Header.Name)
	}
	if gzr.Header.Comment != "test-comment" {
		t.Error("got wrong Name", gzr.Header.Name)
	}
	modTime = modTime.Round(time.Second)
	if gzr.Header.ModTime != modTime {
		t.Errorf("mod time is different: %v %v", gzr.Header.ModTime, modTime)
	}

	var unComp bytes.Buffer
	_, err = io.Copy(&unComp, gzr)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(data, unComp.Bytes()) {
		t.Error("data not same.")
	}

	err = gzr.Close()
	if err != nil {
		t.Error(err)
	}
}
