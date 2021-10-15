# Dropserver

Dropserver is an application platform for your personal web services.

See [Dropserver.org](https://Dropserver.org) for details on the project.

## The Code

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

### Notes on Go Code Organization

Although it goes against Go's recommendations I favor lots of small packages.

The organization of the Go code is inspired by Ben Johnson's [Standard Package Layout](https://www.gobeyond.dev/standard-package-layout/). I took the ideas and evolved them to fit my needs.

Specifically, Dropserver is currently made up of two executables that share a lot of functionality, and I foresee more as the project develops. For this reason it's important to maximize the reusability of packages. This is accomplished as follows:

- If exporting a function works well enough, then do that
- For more demanding situations packages export structs. Any dependency is a field of type `interface` such that the package never imports the packages that it needs, they are injected at compile time.
- Types that are passed between different packages (as function parameters or return types) are defined in a global `domain.go`. Each package only needs to import `domain` to speak the same "language" as the rest of the code.
- The main command packages import and initialize each struct, setting fields to other initialized structs such that the interfaces are satisfied.
- The `testmocks` package generates mocks for all packages that need to be used in testing.
- To prevent nil pointers at interfaces a `checkinject` utility can verify that each required dependency is indeed non-nil. It can also map out the dependency tree (wip).

The result of this scheme is that `ds-dev` (which is a cut-down and tweaked version of `ds-host`) exists with a minimal codebase strictly focused on the parts that are not common with `ds-host`.

### Building

If you want to know how to build locally, please have a look at the Github actions that build the release. This will give you a pretty good idea of the steps you need to take.

### Tips For VSCode Users

I develop in [Visual Studio Code](https://code.visualstudio.com/) therefore some config files in this repo are dependent on VSCode and a few extensions.

I tried to minimize the dependence on VSCode to work effectively on this project, but if there is something that can be done to make it easier for you to work in your preferred editor, please file an issue.

Having said that, here are some extensions that are helpful when working on this repo in VSCode:

- [Go](https://marketplace.visualstudio.com/items?itemName=golang.Go)
- [Deno](https://marketplace.visualstudio.com/items?itemName=denoland.vscode-deno)
- [Vetur](https://marketplace.visualstudio.com/items?itemName=octref.vetur) for Vue Single File Components


## Status of the Project

At this point a good chunk of the project is functional. You can upload app code, create appspaces, migrate, add users, and use the appspace with other users.

However:

- **Not secure at all** (yet). Don't use app code you don't trust. Don't put it up on the public internet unless it's completely isolated in a VM and don't put data on there you can't afford to have stolen.
- Leaks like a sieve. Memory 📈 goroutines 📈
- Lots of missing or half-baked functionality.
- APIs that the apps use are going to change a lot

Code quality is variable. Some parts are OK, lots are pretty shoddy. Sorry.

There is decent code coverage of the Go code (for a project that is nowhere near 1.0). Some of the Go tests are flaky, but I am working on fixing those.

There is little to no test coverage for frontend code, and `denosandboxcode` coverage is sparse.

## Contributing

Contributions are welcome. However given the early stage of this project please start by opening an issue with your proposed contribution.

# License

TODO