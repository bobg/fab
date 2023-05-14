package fab

import "embed"

//go:embed *.go go.* driver.go.tmpl golang/*.go proto/*.go sqlite/*.go sqlite/*.sql ts/*.go
var embeds embed.FS

//go:embed driver.go.tmpl
var driverStr string
