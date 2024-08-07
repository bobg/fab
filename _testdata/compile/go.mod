module q

go 1.20

require github.com/bobg/fab v0.48.4

require (
	github.com/benbjohnson/clock v1.3.0
	github.com/bobg/errors v0.10.0
	github.com/bobg/go-generics/v2 v2.1.2
	github.com/bobg/tsdecls v0.1.0
	github.com/bradleyjkemp/cupaloy/v2 v2.8.0
	github.com/davecgh/go-spew v1.1.1
	github.com/gibson042/canonicaljson-go v1.0.3
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/otiai10/copy v1.7.0
	golang.org/x/mod v0.18.0
	golang.org/x/tools v0.22.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bobg/go-generics v1.5.0 // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/exp v0.0.0-20230213192124-5e25df0256eb // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
)

replace github.com/bobg/fab => ../fab
