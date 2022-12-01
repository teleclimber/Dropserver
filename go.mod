module github.com/teleclimber/DropServer

go 1.16

require (
	github.com/Storytel/gomock-matchers v1.3.0
	github.com/blang/semver/v4 v4.0.0
	github.com/bradfitz/iter v0.0.0-20191230175014-e8f45d346db8
	github.com/caddyserver/certmagic v0.17.2
	github.com/dlclark/regexp2 v1.7.0 // indirect
	github.com/fsnotify/fsnotify v1.5.4
	github.com/go-chi/chi/v5 v5.0.7
	github.com/go-chi/docgen v1.2.0
	github.com/go-playground/validator/v10 v10.11.1
	github.com/golang/mock v1.6.0
	github.com/google/go-cmp v0.5.9
	github.com/google/uuid v1.3.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/otiai10/copy v1.7.0
	github.com/prometheus/client_golang v1.13.0
	github.com/soongo/path-to-regexp v1.6.4
	github.com/teleclimber/twine-go v0.1.2
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be
	golang.org/x/sys v0.0.0-20220927170352-d9d178bc13c6
	gopkg.in/validator.v2 v2.0.1
)

// replace github.com/teleclimber/twine-go => ../twine-go

// replace github.com/mholt/acmez => ../acmez
