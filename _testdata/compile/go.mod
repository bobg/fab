module q

go 1.20

require github.com/bobg/fab v0.29.0

require (
	github.com/benbjohnson/clock v1.3.0
	github.com/bobg/errors v0.10.0
	github.com/bobg/go-generics/v2 v2.1.1
	github.com/bobg/tsdecls v0.1.0
	github.com/gibson042/canonicaljson-go v1.0.3
	github.com/mattn/go-shellwords v1.0.12
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/otiai10/copy v1.7.0
	github.com/pressly/goose/v3 v3.6.1
	golang.org/x/mod v0.6.0
	golang.org/x/tools v0.2.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bobg/go-generics v1.5.0 // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/exp v0.0.0-20230213192124-5e25df0256eb // indirect
	golang.org/x/sync v0.0.0-20220819030929-7fc1605a5dde // indirect
	golang.org/x/sys v0.1.0 // indirect
)

replace github.com/bobg/fab => ../fab
