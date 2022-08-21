package fab

import "embed"

//go:embed *.go go.* driver.go.tmpl
var embeds embed.FS

var driverStr string

func init() {
	driverBytes, err := embeds.ReadFile("driver.go.tmpl")
	if err != nil {
		panic(err)
	}
	driverStr = string(driverBytes)
}
