package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type tempDir struct {
	dir     string
	cleanup bool
}

func setupTempDir() tempDir {
	var td tempDir
	var err error

	if envDir := os.Getenv("DS_DEV_TEMPDIR"); envDir != "" {
		td.dir, err = filepath.Abs(envDir)
		if err != nil {
			panic(fmt.Sprintf("failed to resolve DS_DEV_TEMPDIR: %v", err))
		}
		entries, err := os.ReadDir(td.dir)
		if err != nil {
			panic(fmt.Sprintf("failed to read DS_DEV_TEMPDIR %q: %v", td.dir, err))
		}
		if len(entries) > 0 {
			panic(fmt.Sprintf("DS_DEV_TEMPDIR %q is not empty", td.dir))
		}
	} else {
		td.dir, err = os.MkdirTemp("", "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create temporary directory: %v\nYou can set DS_DEV_TEMPDIR to an empty directory to use as the temp dir.\n", err)
			panic(err)
		}
		td.cleanup = true
	}

	// temp dirs are sometimes symlinks to a dir, which trips up our CWD evaluations, particularly in Deno
	// https://github.com/denoland/deno/issues/22309
	td.dir, err = filepath.EvalSymlinks(td.dir)
	if err != nil {
		panic(err)
	}

	fmt.Println("Temp dir: " + td.dir)

	return td
}

func (td tempDir) cleanupDir() {
	if td.cleanup {
		os.RemoveAll(td.dir)
	}
}
