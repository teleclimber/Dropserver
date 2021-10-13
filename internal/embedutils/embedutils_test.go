package embedutils

import (
	"embed"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/teleclimber/DropServer/internal/embedutils/testfiles"
)

//go:embed testfiles
var testFiles embed.FS

func TestFileToDisk(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = FileToDisk(testFiles, "testfiles/dir2/world.txt", dir)
	if err != nil {
		t.Error(err)
	}

	f, err := ioutil.ReadFile(filepath.Join(dir, "world.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(f) != "WORLD" {
		t.Error("expected WORLD in file")
	}
}

func TestDirToDisk(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = DirToDisk(testFiles, "testfiles", dir)
	if err != nil {
		t.Error(err)
	}

	f, err := ioutil.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(f) != "HELLO" {
		t.Error("expected HELLO in file")
	}

	f, err = ioutil.ReadFile(filepath.Join(dir, "dir2", "world.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(f) != "WORLD" {
		t.Error("expected WORLD in file")
	}
}

func TestStarDirToDisk(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = DirToDisk(testfiles.TestFilesStar, ".", dir)
	if err != nil {
		t.Error(err)
	}

	f, err := ioutil.ReadFile(filepath.Join(dir, "hello.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(f) != "HELLO" {
		t.Error("expected HELLO in file")
	}

	f, err = ioutil.ReadFile(filepath.Join(dir, "dir2", "world.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(f) != "WORLD" {
		t.Error("expected WORLD in file")
	}
}
