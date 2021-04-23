package zipfns

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZipFiles(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")

	os.Mkdir(filesDir, 0700)
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
	for _, f := range r.File {
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

	if count != len(files) {
		t.Error("we didn't get the right number of files in archive")
	}
}

func TestUnzip(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")
	unzipDir := filepath.Join(dir, "unzipped")

	os.Mkdir(filesDir, 0700)
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

	err = Zip(filesDir, zipFile)
	if err != nil {
		t.Error(err)
	}

	err = Unzip(zipFile, unzipDir)
	if err != nil {
		t.Error(err)
	}

	for n, c := range files {
		content, err := ioutil.ReadFile(filepath.Join(unzipDir, n))
		if err != nil {
			t.Error(err)
		}
		if string(content) != c {
			t.Error("wrong content in file")
		}
	}
}

//TestZipSlip inspired by https://gist.github.com/virusdefender/f8c127ab420942dd99acbf269aca9cc7
func TestZipSlip(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	filesDir := filepath.Join(dir, "files")
	zipFile := filepath.Join(dir, "test.zip")
	unzipDir := filepath.Join(dir, "unzipped")

	os.Mkdir(filesDir, 0700)
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

	err = Unzip(zipFile, unzipDir)
	if err == nil {
		t.Error("expected error from Unzip")
	}

	// check legit file is in unzip
	content, err := ioutil.ReadFile(filepath.Join(unzipDir, "hello.txt"))
	if err != nil {
		t.Error(err)
	}
	if string(content) != files["hello.txt"] {
		t.Error("wrong content in file")
	}

	// check evil file is NOT there
	_, err = ioutil.ReadFile(filepath.Join(dir, "evil.txt"))
	if err == nil {
		t.Error("expected error reading evil file")
	}
	_, err = ioutil.ReadFile(filepath.Join(dir, "forbidden", "evil.txt"))
	if err == nil {
		t.Error("expected error reading forbidden/evil file")
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
