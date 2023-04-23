package fab

import "embed"

//go:embed *.go go.* driver.go.tmpl deps/*.go proto/*.go sqlite/*.go sqlite/migrations/*.sql ts/*.go
var embeds embed.FS

//go:embed driver.go.tmpl
var driverStr string
