package fab

import "embed"

//go:embed *.go go.* driver.go.tmpl deps/*.go rules/*.go sqlite/*.go sqlite/migrations/*.sql
var embeds embed.FS

//go:embed driver.go.tmpl
var driverStr string
