package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type AppPackager struct {
	AppGetter interface {
		Reprocess(userID domain.UserID, appID domain.AppID, locationKey string) (domain.AppGetKey, error)
		SubscribeKey(key domain.AppGetKey) (domain.AppGetEvent, <-chan domain.AppGetEvent)
		GetResults(key domain.AppGetKey) (domain.AppGetMeta, bool)
		DeleteKeyData(key domain.AppGetKey)
	}
	AppFilesModel interface {
		ReadManifest(string) (domain.AppVersionManifest, error)
	}
}

func (p *AppPackager) PackageApp(appDir, outDir string, base string) {
	checkOutputDir(outDir)

	results := p.loadAppData()
	if len(results.Errors) != 0 {
		for _, e := range results.Errors {
			fmt.Println(e)
		}
		fmt.Println("Packaging failed. Please fix the errors above and try again.")
		os.Exit(1)
	}

	if len(results.Warnings) != 0 {
		for k, w := range results.Warnings {
			fmt.Printf("Warning: %v: %s\n", k, w)
		}
	}

	// Still to do:
	// - lib/API version
	// - code state? No. Later
	// - Find License file, and get SPDX value?
	// - size (get from tar?)

	manifest, err := p.AppFilesModel.ReadManifest("")
	if err != nil {
		fmt.Println("Error reading app manifest file: ", err)
		os.Exit(1)
	}
	releaseDate := time.Now().Format("2006-01-02")
	manifest.ReleaseDate = releaseDate
	results.VersionManifest.ReleaseDate = releaseDate

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		fmt.Println("Error generating manifest JSON: ", err)
		os.Exit(1)
	}

	fileList, err := GetFileList(appDir)
	if err != nil {
		fmt.Println("Error creating list of app files: ", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	err = tarFiles(&buf, appDir, fileList, manifestBytes)
	if err != nil {
		fmt.Println("Error creating tar archive of app files: ", err)
		os.Exit(1)
	}

	base = fmt.Sprintf("%s-%s", base, string(results.VersionManifest.Version))
	appFd, err := getAppFile(outDir, base)
	if err != nil {
		fmt.Println("Error creating package file: ", err)
		os.Exit(1)
	}
	defer appFd.Close()

	err = gzipArchive(appFd, buf.Bytes(), results.VersionManifest.Name, "Dropserver app package created using ds-dev "+cmd_version, time.Now())
	if err != nil {
		fmt.Println("Error gzipping: ", err)
		os.Exit(1)
	}

	err = writeManifestFile(results.VersionManifest, outDir, base)
	if err != nil {
		fmt.Println("Error creating manifest file: ", err)
		os.Exit(1)
	}
}

func (p *AppPackager) loadAppData() domain.AppGetMeta {
	appGetKey, err := p.AppGetter.Reprocess(ownerID, appID, "")
	if err != nil {
		panic(err)
	}

	lastEvent, appGetCh := p.AppGetter.SubscribeKey(appGetKey)
	if lastEvent.Done || appGetCh == nil {
		return p.getResults(appGetKey)
	}

	rChan := make(chan domain.AppGetMeta, 1)
	done := false
	for e := range appGetCh {
		if e.Done {
			if !done {
				fmt.Println("Done processing app")
				go func() { // have to do this to prevent deadlock
					r := p.getResults(appGetKey)
					rChan <- r
				}()
			}
			done = true
		} else {
			fmt.Println(e.Step)
		}
	}
	return <-rChan
}

func (p *AppPackager) getResults(appGetKey domain.AppGetKey) domain.AppGetMeta {
	results, ok := p.AppGetter.GetResults(appGetKey)
	if !ok {
		panic("no appGetKey. This is a bug in ds-dev.")
	}
	p.AppGetter.DeleteKeyData(appGetKey)
	return results
}

func gzipArchive(w io.Writer, archive []byte, name, comment string, modTime time.Time) error {
	gzw := gzip.NewWriter(w)
	gzw.Name = name
	gzw.Comment = comment
	gzw.ModTime = modTime.Round(time.Second)
	_, err := gzw.Write(archive)
	if err != nil {
		return err
	}
	gzw.Close()
	return nil
}

func tarFiles(w io.Writer, baseDir string, list []FileListFile, manifestBytes []byte) error {
	tw := tar.NewWriter(w)
	defer tw.Close()
	var src io.ReadCloser
	var hdr tar.Header
	var err error
	foundManifest := false
	for _, f := range list {
		if f.Ignore {
			continue
		}
		if f.Name == "dropapp.json" {
			hdr = tar.Header{
				Name:    f.Name,
				Mode:    0644,
				Size:    int64(len(manifestBytes)),
				ModTime: f.ModTime}
			src = io.NopCloser(bytes.NewBuffer(manifestBytes))
			foundManifest = true
		} else {
			hdr = tar.Header{
				Name:    f.Name,
				Mode:    0644,
				Size:    f.Size,
				ModTime: f.ModTime}
			fullPath := filepath.Join(baseDir, f.Name)
			src, err = os.Open(fullPath)
			if err != nil {
				return err
			}
		}
		err = tw.WriteHeader(&hdr)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, src)
		src.Close()
		if err != nil {
			return err
		}
	}
	if !foundManifest {
		return errors.New("did not find manifest")
	}
	return nil
}

type FileListFile struct {
	Name    string
	Size    int64
	ModTime time.Time
	IsDir   bool
	Ignore  bool
}

func GetFileList(appDir string) ([]FileListFile, error) {
	appDir = filepath.Clean(appDir) + string(filepath.Separator)
	fileList := make([]FileListFile, 0)
	err := filepath.Walk(appDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			// failure accessing a path...
			return err
		}
		fName := info.Name()
		relName := strings.TrimPrefix(path, appDir)
		if info.IsDir() {
			if skip(fName) {
				fileList = append(fileList, FileListFile{Name: relName, IsDir: true, Ignore: true})
				return filepath.SkipDir
			}
		} else {
			fileList = append(fileList, FileListFile{
				Name:    relName,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				Ignore:  skip(fName)})
		}
		return nil
	})
	return fileList, err
}

// var skipFileNames = []string{} // what should go in there?
func skip(name string) bool {
	// for _, s := range skipFileNames {
	// 	if s == name {
	// 		return true
	// 	}
	// }
	// filter out "dot" files and directories
	if strings.HasPrefix(name, ".") {
		return true
	}

	return false
}

func checkOutputDir(outDir string) {
	info, err := os.Stat(outDir)
	if err == os.ErrNotExist {
		fmt.Println("Output dir does not exist: " + outDir)
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("Error opening output dir: ", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Println("Output Directory is not a directory: " + outDir)
		os.Exit(1)
	}
}

func getAppFile(outDir, base string) (*os.File, error) {
	fullPath := filepath.Join(outDir, base+".tar.gz")
	fd, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return nil, errors.New("File already exists: " + fullPath)
	}
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func writeManifestFile(manifest domain.AppVersionManifest, outDir, base string) error {
	manifestBytes, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return fmt.Errorf("error creating manifest JSON: %w", err)
	}
	fullPath := filepath.Join(outDir, base+".json")
	fd, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return errors.New("File already exists: " + fullPath)
	}
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.Write(manifestBytes)
	if err != nil {
		return err
	}
	return nil
}
