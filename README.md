# Dropserver

Dropserver is an application platform for your personal web services.

See [Dropserver.org](https://Dropserver.org) for details on the project.

# The Code

Dropserver is written mainly in Go, but uses [Deno](https://deno.land) as the the sandbox for app code. Deno is not packaged with the DS executables, instead it must be installed separately.

This repository builds two executables:

- `ds-host` is the complete server that can run apps in appspaces and responds to requests directed at those appspaces. It should run on a cloud VM or on a home server. Linux/x86_64 only.
- `ds-dev` is intended to run locally and is used to run Dropserver applications while they are being developed. Builds for Linux and Mac are available.

High level directories of Dropserver repo:

- `/cmd/ds-host` is where most of the Go code lives
- `/cmd/ds-dev` contains the `ds-dev` Go code. It pulls in a lot of packages from the `ds-host` sibling directory
- `/denosandboxcode` is [Deno](https://deno.land) Typescript code that runs in each sandbox
- `/frontend-ds-host` and `frontend-ds-dev` are the frontend code (Vue 3)
- `/internal` additional Go code that is not specific to Dropserver
- `/scripts` local build scripts for `ds-host` and `ds-dev`

## Notes on Go Code Organization

Although it goes against Go's recommendations I favor lots of small packages.

The organization of the Go code is inspired by Ben Johnson's [Standard Package Layout](https://www.gobeyond.dev/standard-package-layout/). I took the ideas and evolved them to fit the needs of this project.

Specifically, Dropserver is currently made up of two executables that share a lot of functionality, and I foresee more as the project develops. For this reason it's important to maximize the reusability of packages. This is accomplished as follows:

- If exporting a function works well enough, then do that
- For more demanding situations packages export structs. Any dependency is a field of type `interface` such that the package never imports the packages that it needs, they are injected at compile time.
- Types that are passed between different packages (as function parameters or return types) are defined in a global `domain.go`. Each package only needs to import `domain` to speak the same "language" as the rest of the code.
- The main command packages import and initialize each struct, setting fields to other initialized structs such that the interfaces are satisfied.
- The `testmocks` package generates mocks for all packages that need to be used in testing.
- To prevent nil pointers at interfaces a `checkinject` utility can verify that each required dependency is indeed non-nil. It can also map out the dependency tree (wip).

The result of this scheme is that `ds-dev` (which is a cut-down and tweaked version of `ds-host`) exists with a minimal codebase strictly focused on the parts that are not common with `ds-host`.

## Building

If you want to know how to build locally, please have a look at the Github actions that build the release. This will give you a pretty good idea of the steps you need to take.

## Tips For VSCode Users

I develop in [Visual Studio Code](https://code.visualstudio.com/) therefore some config files in this repo are dependent on VSCode and a few extensions.

I tried to minimize the dependence on VSCode to work effectively on this project, but if there is something that can be done to make it easier for you to work in your preferred editor, please file an issue.

Having said that, here are some extensions that are helpful when working on this repo in VSCode:

- [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go)
- [Deno](https://marketplace.visualstudio.com/items?itemName=denoland.vscode-deno)
- [Vetur](https://marketplace.visualstudio.com/items?itemName=octref.vetur) for Vue Single File Components

# Installing and Running

## *Warning: Dropserver is very new and **probably has security holes** right now!*

Obtain the latest release from the [Releases](https://github.com/teleclimber/Dropserver/releases) page and unzip.

You should have [Deno](https//deno.land) installed and available from wherever you'll be running `ds-host` or `ds-dev`.

## ds-dev

`ds-dev` is used when developing an app locally. It watches your app files and restarts the sandbox as needed.

Read about using `ds-dev` in the [Dropserver Docs](https://dropserver.org/docs/ds-dev/).

## ds-host

`ds-host` is the full Dropserver and is intended to serve your appspaces in use.

Running `ds-host` takes a bit more work:

1. You'll need a domain (or subdomain) and point it and its subdomains (wildcard) to your server's IP
2. Create a TLS cert for that domain and all its subdomains (wildcard). You can do this with [Letsencrypt](https://www.digitalocean.com/community/tutorials/how-to-create-let-s-encrypt-wildcard-certificates-with-certbot)
3. Create an empty data directory, let's say it is `/home/dev/ds-data`
4. Create and empty directory for sockets, say `/home/dev/ds-sockets`
5. Create a JSON config file at `/home/dev/ds-config.json` (see below)
6. Run `ds-host -config=/home/dev/ds-config.json -migrate` to create the DB and the admin user
7. Run `ds-host -config=/home/dev/ds-config.json`

Beyond that, I recommend running `ds-host` as a systemd service, and enabling cgroups to monitor appspace resource usage (see below).

### ds-config.json

If you are running `ds-host` behind a reverse proxy with SSL termination (recommended) your config might look something like this:

```
{
	"data-dir": "/home/dev/ds-data",
	"server": {
		"port": 5050,
		"host": "example.com",
		"ssl-cert": "",
		"ssl-key": ""
	},
	"port-string":"",
	"subdomains":{
		"user-accounts": "dropid",
		"static-assets": "static"
	},
	"sandbox":{
		"sockets-dir": "/home/dev/ds-sockets",
		"use-cgroups": false
	}
}
```

Note that with the chosen subdomains, you log in at https://dropid.example.com.

If you are experimenting on a local network, you might try a configuration like this one:

```
{
	"data-dir": "/home/dev/ds-data",
	"server": {
		"port": 5050,
		"host": "example.com",
		"ssl-cert": "/home/dev/ssl/example_com.crt",
		"ssl-key": "/home/dev/ssl/example_com.key"
	},
	"port-string":":5050",
	"trust-cert": "/home/dev/ssl/rootSSL.pem",
	"subdomains":{
		"user-accounts": "dropid",
		"static-assets": "static"
	},
	"sandbox":{
		"sockets-dir": "/home/dev/ds-sockets",
		"use-cgroups": false
	}
}
```

Notes:
- site will be reachable at https://dropid.example.com:5050 (Set your local DNS server accordingly)
- there would be no reverse proxy in this scenario, and `ds-host` does the SSL termination
- `port-string` is used when generating links in case your installation is reachable on a port other than :80 or :443. In this case it's `:5050`.
- `trust-cert` lets you specify the root SSL cert of your root CA

You can also skip the whole SSL thing for local experimentation like this:

```
{
	"data-dir": "/home/dev/ds-data",
	"server": {
		"port": 5050,
		"host": "mydomain.com",
		"ssl-cert": "",
		"ssl-key": ""
	},
	"no-tls": true,
	"port-string":":5050",
	"subdomains":{
		"user-accounts": "dropid",
		"static-assets": "static"
	},
	"sandbox":{
		"sockets-dir": "/home/dev/ds-sockets",
		"use-cgroups": false
	}
}
```

### CGroups

By default `ds-host` uses cgroups (version 2) to measure and control resources used by appspaces (this is a work in progress). 

**Important:** To use cgroups, `ds-host` must run in a delegatable cgroup. This is typically accomplished by having systemd run ds-host as a service, and setting `Delegate=true` in the service config.

The default config for cgroups looks like this:

```
{
	...
	"sandbox": {
		...
		"use-cgroups": true,
		"cgroup-mount": "/sys/fs/cgroup",
		"memory-high-mb": 512
	}
}
```

- `use-cgroups` set to false if you can not or do not want to deal with cgroups.
- `cgroup-mount`: the location of your system's cgroups.
- `memory-high-mb`: the `memory.high` value for the cgroup that contains all the sandbox cgroups, in megabytes.

## Leftovers App

Clone and build the [Leftovers app](https://github.com/teleclimber/Leftovers) to have something to play with once you have Dropserver running.

# Status of the Project

At this point a good chunk of the project is functional. You can upload app code, create appspaces, migrate, add users, and use the appspace with other users.

However:

- **Not secure at all** (yet). Don't use app code you don't trust. Don't put it up on the public internet unless it's completely isolated in a VM and don't put data on there you can't afford to have stolen.
- Leaks like a sieve. Memory 📈 goroutines 📈
- Lots of missing or half-baked functionality.
- APIs that the apps use are going to change a lot

Code quality is variable. Some parts are OK, some are pretty shoddy. Sorry.

There is decent code coverage of the Go code (for a project that is nowhere near 1.0).

There is little to no test coverage for frontend code, and `denosandboxcode` coverage is sparse.

# Contributing

Contributions are welcome. However given the early stage of this project please start by opening an issue with your proposed contribution.

# License

TODO