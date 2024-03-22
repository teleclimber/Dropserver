package sandbox

import (
	"path/filepath"
	"strings"

	"github.com/teleclimber/DropServer/cmd/ds-host/domain"
)

type pathData struct {
	host       string // the original host path
	bwrap      string // the same file or dir from inside bwrap sandbox
	bwrapRead  bool   // whether readable inside bwrap sandbox
	bwrapWrite bool   // whether writable inside bwrap sandbox
	importMap  bool   // Whether to include in Deno import-map
	denoRead   bool   // whether readbable inside Deno sandbox
	denoWrite  bool   // whether writable inside Deno sandbox
}

func (d *pathData) hostPath() string {
	return d.host
}

func (d *pathData) sandboxPath(useBwrap bool) string {
	if useBwrap {
		return d.bwrap
	}
	return d.host
}

// Could we make a struct that embodies all variations and transformations that path go through?
type paths struct {
	appLoc      string
	appspaceLoc string // if "" then no appspace.
	sockets     string // set by sandbox while starting up

	Config           *domain.RuntimeConfig
	AppLocation2Path interface {
		Meta(string) string
		Files(string) string
		DenoDir(string) string
	}
	AppspaceLocation2Path interface {
		Base(string) string
		Data(string) string
		Files(string) string
		Avatars(string) string
		DenoDir(string) string
	}

	importMapExtras map[string]string

	pDatas    []pathData
	pDatasMap map[string]pathData
}

func (p *paths) init() {
	p.pDatas = make([]pathData, 0)
	p.pDatasMap = make(map[string]pathData)

	deno := "deno"
	if p.Config.Sandbox.UseBubblewrap {
		deno = p.Config.Exec.DenoFullPath
	}

	p.add("deno", pathData{
		host:      deno,
		bwrap:     "/deno",
		bwrapRead: true,
	})

	p.add("bootstrap", pathData{
		host:      filepath.Join(p.AppLocation2Path.Meta(p.appLoc), "bootstrap.js"),
		bwrap:     "/bootstrap.js",
		bwrapRead: true,
		importMap: true, // apparently it's needed
	})

	p.add("goproxy-cert", pathData{
		host:      filepath.Join(p.Config.Exec.RuntimeFilesPath, "goproxy-ca-cert.pem"),
		bwrap:     "/goproxy-ca-cert.pem",
		bwrapRead: true,
	})

	p.add("sandbox-runner", pathData{
		host:      trailingSlash(p.Config.Exec.SandboxCodePath),
		bwrap:     "/deno-sandbox-runner/",
		bwrapRead: true,
		importMap: true,
	})

	p.add("sockets", pathData{
		host:      p.sockets,
		bwrap:     "/sockets/",
		denoWrite: true,
	})

	p.add("app-files", pathData{
		host:      trailingSlash(p.AppLocation2Path.Files(p.appLoc)),
		bwrap:     "/app-files/",
		importMap: true,
		denoRead:  true,
	})

	if p.appspaceLoc == "" {
		p.add("import-map", pathData{
			host:      filepath.Join(p.AppLocation2Path.Meta(p.appLoc), "import-paths.json"),
			bwrap:     "/import-paths.json",
			bwrapRead: true,
		})
		p.add("deno-dir", pathData{
			host:       p.AppLocation2Path.DenoDir(p.appLoc),
			bwrap:      "/deno-dir/",
			bwrapWrite: true,
		})
	} else {
		p.add("import-map", pathData{
			host:      filepath.Join(p.AppspaceLocation2Path.Base(p.appspaceLoc), "import-paths.json"),
			bwrap:     "/import-paths.json",
			bwrapRead: true,
		})
		p.add("deno-dir", pathData{
			host:       p.AppspaceLocation2Path.DenoDir(p.appspaceLoc),
			bwrap:      "/deno-dir/",
			bwrapWrite: true,
		})
		p.add("appspace-data", pathData{
			host:  trailingSlash(p.AppspaceLocation2Path.Data(p.appspaceLoc)),
			bwrap: "/appspace-data/",
		})
		p.add("appspace-files", pathData{
			host:      trailingSlash(p.AppspaceLocation2Path.Files(p.appspaceLoc)),
			bwrap:     "/appspace-data/files/",
			importMap: true,
			denoWrite: true,
		})
		p.pDatas = append(p.pDatas, pathData{
			host:     trailingSlash(p.AppspaceLocation2Path.Avatars(p.appspaceLoc)),
			bwrap:    "/appspace-data/avatars/",
			denoRead: true,
		})
	}
}

func (p *paths) add(k string, d pathData) {
	p.pDatasMap[k] = d
	p.pDatas = append(p.pDatas, d)
}

func (p *paths) hostPath(k string) string {
	pd, ok := p.pDatasMap[k]
	if !ok {
		panic("trying to get pathKey that doesn't exist in pDatasMap: " + k)
	}
	return pd.hostPath()
}

// sandboxPath returns a path that works from inside the sadnbox
func (p *paths) sandboxPath(k string) string {
	pd, ok := p.pDatasMap[k]
	if !ok {
		panic("trying to get pathKey that doesn't exist in pDatasMap: " + k)
	}
	return pd.sandboxPath(p.Config.Sandbox.UseBubblewrap)
}
func (p *paths) denoImportMap() ImportPaths {
	im := ImportPaths{
		Imports: map[string]string{
			"/":  "/dev/null/", // Defeat imports from outside the app dir. See:
			"./": "./",         // https://github.com/denoland/deno/issues/6294#issuecomment-663256029
		}}

	for _, pd := range p.pDatas {
		if pd.importMap {
			entry := pd.sandboxPath(p.Config.Sandbox.UseBubblewrap)
			// here check that entry is not somehow equivalent to "/"
			im.Imports[entry] = entry
		}
	}

	if p.importMapExtras != nil {
		for k, v := range p.importMapExtras {
			im.Imports[k] = v
		}
	}

	return im
}

func (p *paths) getBwrapPathMaps() []string {
	elems := make([]string, 0)
	for _, e := range p.pDatas {
		if e.bwrapWrite || e.denoWrite {
			elems = append(elems, "--bind", e.hostPath(), e.sandboxPath(true))
		} else if e.bwrapRead || e.denoRead {
			elems = append(elems, "--ro-bind", e.hostPath(), e.sandboxPath(true))
		}
	}
	return elems
}

func (p *paths) denoAllowRead() string {
	elems := make([]string, 0)
	for _, e := range p.pDatas {
		if e.denoRead || e.denoWrite {
			elems = append(elems, e.sandboxPath(p.Config.Sandbox.UseBubblewrap))
		}
	}
	return strings.Join(elems, ",")
}

func (p *paths) denoAllowWrite() string {
	elems := make([]string, 0)
	for _, e := range p.pDatas {
		if e.denoWrite {
			elems = append(elems, e.sandboxPath(p.Config.Sandbox.UseBubblewrap))
		}
	}
	return strings.Join(elems, ",")
}

func trailingSlash(p string) string {
	if strings.HasSuffix(p, "/") {
		return p
	}
	return p + "/"
}
