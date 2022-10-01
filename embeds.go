package fab

import "embed"

//go:embed *.go go.* driver.go.tmpl deps sqlite
var embeds embed.FS

//go:embed driver.go.tmpl
var driverStr string
