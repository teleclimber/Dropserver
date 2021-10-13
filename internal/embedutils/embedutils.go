package embedutils

import (
	"embed"
	"io"
	"os"
	"path"
	"path/filepath"
)

// DirToDisk copies the contents of the embedFS
// to disk at toDir.
func DirToDisk(embedFS embed.FS, embedDir, toDir string) error {
	os.MkdirAll(toDir, 0744)

	entries, err := embedFS.ReadDir(embedDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryName := entry.Name()
		if entry.IsDir() {
			err = DirToDisk(embedFS, filepath.Join(embedDir, entryName), filepath.Join(toDir, entryName))
			if err != nil {
				return err // maybe wrap it so we know what file caused the fail
			}
		} else {
			err = FileToDisk(embedFS, filepath.Join(embedDir, entryName), toDir)
			if err != nil {
				return err // maybe wrap it so we know what file caused the fail
			}
		}
	}
	return nil
}

// adapted from
// https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
// By Roland Singer [roland.singer@desertbit.com]
func FileToDisk(embedFS embed.FS, embedPath, toDir string) (err error) {
	in, err := embedFS.Open(embedPath)
	if err != nil {
		return
	}
	defer in.Close()

	outPath := filepath.Join(toDir, path.Base(embedPath))

	out, err := os.Create(outPath)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	err = os.Chmod(outPath, 0644)
	if err != nil {
		return
	}

	return
}
