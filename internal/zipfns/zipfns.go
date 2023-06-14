package zipfns

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

// zipFiles puts all the files found at source dir into
// zip file. See:
// https://stackoverflow.com/a/63233911
func Zip(sourceDir, destFile string) error {
	file, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			_, err := zipWriter.Create(relPath + string(filepath.Separator))
			if err != nil {
				return err
			}
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(sourceDir, walker)
	if err != nil {
		return err
	}

	return nil
}

// https://stackoverflow.com/a/24792688
// TODO: handle symbolic links
// TODO: come up with better permissions
func Unzip(src, dest string, maxSize int64) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	dest = filepath.Clean(dest)

	err = os.MkdirAll(dest, 0755)
	if err != nil {
		return err
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File, max int64) (int64, error) {
		rc, err := f.Open()
		if err != nil {
			return 0, err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if path == dest {
			return 0, nil //skip the root path of the zip
		}

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, dest+string(os.PathSeparator)) {
			return 0, fmt.Errorf("illegal file path: %s", path)
		}

		actualSize := int64(0)
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(path, 0755)
			if err != nil {
				return 0, err
			}
		} else {
			err = os.MkdirAll(filepath.Dir(path), 0755)
			if err != nil {
				return 0, err
			}
			destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return 0, err
			}
			defer func() {
				if err := destFile.Close(); err != nil {
					panic(err)
				}
			}()

			if f.FileInfo().Size() > max {
				return 0, domain.ErrStorageExceeded
			}
			limitedR := limitedReader{R: rc, N: max}
			actualSize, err = io.Copy(destFile, &limitedR)
			if err != nil {
				return actualSize, err
			}
		}
		return actualSize, nil
	}

	//maxSize := int64(1 << 30) // hard-coded to 1 gig for now.
	size := int64(0)
	for _, f := range r.File {
		s, err := extractAndWriteFile(f, maxSize-size)
		if err != nil {
			return err
		}
		size += s
		if size > maxSize {
			return domain.ErrStorageExceeded
		}
	}

	return nil
}

// limitedReader is a copy of io.LimitedReader
// except it returns domain.ErrStorageExceeded
type limitedReader struct {
	R io.Reader // underlying reader
	N int64     // max bytes remaining
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, domain.ErrStorageExceeded
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}
