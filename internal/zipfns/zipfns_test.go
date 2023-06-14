package zipfns

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

func TestZipFiles(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")

	os.Mkdir(filesDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	files["hello.txt"] = "Hello World!"
	files["dir1/dir2/file2.txt"] = "Hello agagin!"
	err = testCreateFiles(filesDir, files)
	if err != nil {
		t.Fatal(err)
	}
	emptyDir := "empty/dir"
	err = os.MkdirAll(filepath.Join(filesDir, emptyDir), 0755)
	if err != nil {
		t.Error(err)
	}

	err = Zip(filesDir, zipFile)
	if err != nil {
		t.Error(err)
	}

	r, err := zip.OpenReader(zipFile)
	if err != nil {
		t.Error(err)
	}
	defer r.Close()

	count := 0
	foundEmptyDir := false
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			if filepath.Clean(f.Name) == emptyDir {
				foundEmptyDir = true
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Error(err)
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(rc)
		fileData := buf.String()

		if fileData != files[f.Name] {
			t.Error("incorrect content")
		}
		rc.Close()
		count++
	}

	if !foundEmptyDir {
		t.Error("did not find our empty dir")
	}
	if count != len(files) {
		t.Error("we didn't get the right number of files in archive")
	}
}

func TestUnzip(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")
	unzipDir := filepath.Join(dir, "unzipped")

	os.Mkdir(filesDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	files["hello.txt"] = "Hello World!"
	files["dir1/dir2/file2.txt"] = "Hello agagin!"
	err = testCreateFiles(filesDir, files)
	if err != nil {
		t.Fatal(err)
	}
	emptyDir := "empty/dir"
	err = os.MkdirAll(filepath.Join(filesDir, emptyDir), 0755)
	if err != nil {
		t.Error(err)
	}

	err = Zip(filesDir, zipFile)
	if err != nil {
		t.Error(err)
	}

	err = Unzip(zipFile, unzipDir, 1<<30)
	if err != nil {
		t.Error(err)
	}

	for n, c := range files {
		content, err := os.ReadFile(filepath.Join(unzipDir, n))
		if err != nil {
			t.Error(err)
		}
		if string(content) != c {
			t.Error("wrong content in file")
		}
	}
	_, err = os.Stat(filepath.Join(unzipDir, emptyDir))
	if os.IsNotExist(err) {
		t.Error("empty dir not in unzipped")
	}
}

// TestZipSlip inspired by https://gist.github.com/virusdefender/f8c127ab420942dd99acbf269aca9cc7
func TestZipSlip(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")
	unzipDir := filepath.Join(dir, "unzipped")

	os.Mkdir(filesDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	files["hello.txt"] = "Hello World!"
	files["../evil.txt"] = "Hello! I'm evil."
	files["../forbidden/evil.txt"] = "Hello! I'm super evil."

	// don't use our own Zip fn, because it may have mitigations against attacks at the zip-creation stage.
	file, err := os.Create(zipFile)
	if err != nil {
		t.Error(err)
	}
	zipWriter := zip.NewWriter(file)
	for n, c := range files {
		f, err := zipWriter.Create(n)
		if err != nil {
			t.Error(err)
		}
		_, err = io.Copy(f, strings.NewReader(c))
		if err != nil {
			t.Error(err)
		}
	}
	zipWriter.Close()
	file.Close()

	err = Unzip(zipFile, unzipDir, 1<<30)
	if err == nil {
		t.Error("expected error from Unzip")
	}

	// check evil file is NOT there
	_, err = os.ReadFile(filepath.Join(dir, "evil.txt"))
	if err == nil {
		t.Error("expected error reading evil file")
	}
	_, err = os.ReadFile(filepath.Join(dir, "forbidden", "evil.txt"))
	if err == nil {
		t.Error("expected error reading forbidden/evil file")
	}
}

// TestUnzipNested verifies that unzipping only unzips one level of zip files.
// Nested Zip files should not be unzipped.
func TestUnzipNested(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	wrapDir := filepath.Join(dir, "wrap")
	zipFile := filepath.Join(dir, "test.zip")
	unzipDir := filepath.Join(dir, "unzipped")

	os.Mkdir(filesDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	files["hello.txt"] = "Hello World!"
	files["dir1/dir2/file2.txt"] = "Hello agagin!"
	err = testCreateFiles(filesDir, files)
	if err != nil {
		t.Fatal(err)
	}
	emptyDir := "empty/dir"
	err = os.MkdirAll(filepath.Join(filesDir, emptyDir), 0755)
	if err != nil {
		t.Error(err)
	}

	err = os.MkdirAll(wrapDir, 0755)
	if err != nil {
		t.Error(err)
	}
	innerZip := filepath.Join(wrapDir, "inner.zip")
	err = Zip(filesDir, innerZip)
	if err != nil {
		t.Error(err)
	}

	err = Zip(wrapDir, zipFile)
	if err != nil {
		t.Error(err)
	}

	err = Unzip(zipFile, unzipDir, 1<<30)
	if err != nil {
		t.Error(err)
	}

	// check that all we have is the inner zip in the unzip:
	stat, err := os.Stat(filepath.Join(unzipDir, "inner.zip"))
	if err != nil {
		t.Error(err)
	}
	if stat.Size() == 0 {
		t.Error("unexpected 0 size for inner zip")
	}
}

func TestUnzipTooLarge(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")
	unzipDir := filepath.Join(dir, "unzipped")

	os.Mkdir(filesDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	files := make(map[string]string)
	files["hello.txt"] = "Hello World!"
	files["dir1/dir2/file2.txt"] = "Hello agagin!"
	err = testCreateFiles(filesDir, files)
	if err != nil {
		t.Fatal(err)
	}
	emptyDir := "empty/dir"
	err = os.MkdirAll(filepath.Join(filesDir, emptyDir), 0755)
	if err != nil {
		t.Error(err)
	}

	err = Zip(filesDir, zipFile)
	if err != nil {
		t.Error(err)
	}

	err = Unzip(zipFile, unzipDir, 10)
	if err != domain.ErrStorageExceeded {
		t.Error("expected Error storage exceeded")
	}
}

func testCreateFiles(dir string, files map[string]string) error {
	for n, c := range files {
		os.MkdirAll(filepath.Join(dir, filepath.Dir(n)), 0700)
		file, err := os.Create(filepath.Join(dir, n))
		if err != nil {
			return err
		}
		file.WriteString(c)
		file.Close()
	}
	return nil
}
