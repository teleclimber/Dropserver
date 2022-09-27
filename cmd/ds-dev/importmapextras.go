package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ImportMapExtras struct {
	SandboxManager interface {
		SetImportMapExtras(importMap map[string]string)
	}
	AppWatcher interface {
		AddDir(dir string)
	}
}

func (m *ImportMapExtras) Init(jsonFile string) {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	var importMapExtras map[string]string
	importMapLoc := *importMapFlag
	if importMapLoc != "" {
		if !filepath.IsAbs(importMapLoc) {
			importMapLoc = filepath.Join(workingDir, importMapLoc)
		}
		contents, err := os.ReadFile(importMapLoc)
		if err != nil {
			panic(err)
		}
		importMapExtras = make(map[string]string)
		err = json.Unmarshal(contents, &importMapExtras)
		if err != nil {
			panic(err)
		}

		m.SandboxManager.SetImportMapExtras(importMapExtras)

		targets := getUniqueTargets(importMapExtras)
		for _, t := range targets {
			m.AppWatcher.AddDir(t)
		}
	}
}

func getUniqueTargets(extras map[string]string) []string {
	ret := make([]string, 0)
	for _, t := range extras {
		found := false
		for _, x := range ret {
			if x == t {
				found = true
				break
			}
		}
		if !found {
			ret = append(ret, t)
		}
	}
	return ret
}
